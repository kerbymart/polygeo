package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kerbymart/polygeo/internal/geo"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type API struct {
	registry *geo.Registry
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type regionListResponse struct {
	Country string   `json:"country"`
	Level   string   `json:"level"`
	Parent  string   `json:"parent,omitempty"`
	Count   int      `json:"count"`
	Results []string `json:"results"`
}

func New(registry *geo.Registry) *echo.Echo {
	api := &API{registry: registry}
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/status", api.status)
	e.GET("/countries", api.countries)
	api.registerCountryRoutes(e, "/countries")
	api.registerCountryRoutes(e, "")
	return e
}

func (a *API) registerCountryRoutes(e *echo.Echo, prefix string) {
	e.GET(prefix+"/:country", a.country)
	e.GET(prefix+"/:country/locate", a.locate)
	e.GET(prefix+"/:country/regions", a.regions)
	e.GET(prefix+"/:country/:level", a.regionsByPath)
	e.GET(prefix+"/:country/:parentLevel/:parent/:childLevel", a.childRegionsByPath)
}

func (a *API) status(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":    "OK",
		"countries": a.registry.Count(),
	})
}

func (a *API) countries(c *echo.Context) error {
	countries := a.registry.Countries()
	return c.JSON(http.StatusOK, map[string]any{
		"count":   len(countries),
		"results": countries,
	})
}

func (a *API) country(c *echo.Context) error {
	country, ok := a.registry.Country(c.Param("country"))
	if !ok {
		return writeError(c, http.StatusNotFound, "country_not_found", "country data package was not found")
	}
	return c.JSON(http.StatusOK, country.Description())
}

func (a *API) regions(c *echo.Context) error {
	level := strings.TrimSpace(c.QueryParam("level"))
	if level == "" {
		return writeError(c, http.StatusBadRequest, "level_required", "the level query parameter is required")
	}
	return a.writeRegions(c, level, strings.TrimSpace(c.QueryParam("parent")))
}

func (a *API) regionsByPath(c *echo.Context) error {
	return a.writeRegions(c, c.Param("level"), strings.TrimSpace(c.QueryParam("parent")))
}

func (a *API) childRegionsByPath(c *echo.Context) error {
	country, ok := a.registry.Country(c.Param("country"))
	if !ok {
		return writeError(c, http.StatusNotFound, "country_not_found", "country data package was not found")
	}
	parentLevel, ok := country.ResolveLevel(c.Param("parentLevel"))
	if !ok {
		return writeError(c, http.StatusNotFound, "level_not_found", "parent administrative level was not found")
	}
	childLevel, ok := country.ResolveLevel(c.Param("childLevel"))
	if !ok {
		return writeError(c, http.StatusNotFound, "level_not_found", "child administrative level was not found")
	}
	if !strings.EqualFold(childLevel.Manifest.ParentLevel, parentLevel.Manifest.ID) {
		return writeError(c, http.StatusBadRequest, "invalid_level_relationship", "the requested child level does not belong to the requested parent level")
	}
	return a.writeRegionsForCountry(c, country, childLevel.Manifest.ID, c.Param("parent"))
}

func (a *API) writeRegions(c *echo.Context, level, parent string) error {
	country, ok := a.registry.Country(c.Param("country"))
	if !ok {
		return writeError(c, http.StatusNotFound, "country_not_found", "country data package was not found")
	}
	return a.writeRegionsForCountry(c, country, level, parent)
}

func (a *API) writeRegionsForCountry(c *echo.Context, country *geo.Country, levelName, parent string) error {
	level, ok := country.ResolveLevel(levelName)
	if !ok {
		return writeError(c, http.StatusNotFound, "level_not_found", "administrative level was not found")
	}
	regions, err := country.Regions(level.Manifest.ID, parent)
	if err != nil {
		return writeError(c, http.StatusBadRequest, "invalid_region_query", err.Error())
	}
	return c.JSON(http.StatusOK, regionListResponse{
		Country: country.Manifest.Code,
		Level:   level.Manifest.ID,
		Parent:  parent,
		Count:   len(regions),
		Results: regions,
	})
}

func (a *API) locate(c *echo.Context) error {
	country, ok := a.registry.Country(c.Param("country"))
	if !ok {
		return writeError(c, http.StatusNotFound, "country_not_found", "country data package was not found")
	}
	latitude, err := parseCoordinate(c.QueryParam("latitude"), "latitude", -90, 90)
	if err != nil {
		return writeError(c, http.StatusBadRequest, "invalid_latitude", err.Error())
	}
	longitude, err := parseCoordinate(c.QueryParam("longitude"), "longitude", -180, 180)
	if err != nil {
		return writeError(c, http.StatusBadRequest, "invalid_longitude", err.Error())
	}
	location, found := country.Locate(longitude, latitude)
	if !found {
		return writeError(c, http.StatusNotFound, "location_not_found", "the coordinate is outside the loaded administrative regions")
	}
	return c.JSON(http.StatusOK, location)
}

func parseCoordinate(raw, name string, minimum, maximum float64) (float64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", name)
	}
	if value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be between %v and %v", name, minimum, maximum)
	}
	return value, nil
}

func writeError(c *echo.Context, status int, code, message string) error {
	if err := c.JSON(status, errorEnvelope{Error: apiError{Code: code, Message: message}}); err != nil {
		return errors.Join(fmt.Errorf("write API error response"), err)
	}
	return nil
}
