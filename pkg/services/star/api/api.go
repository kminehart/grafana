package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/infra/log"
	contextmodel "github.com/grafana/grafana/pkg/services/contexthandler/model"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/star"
	"github.com/grafana/grafana/pkg/web"
)

type API struct {
	starService      star.Service
	dashboardService dashboards.DashboardService
	logger           log.Logger
}

func ProvideApi(
	starService star.Service,
	dashboardService dashboards.DashboardService,
) *API {
	starLogger := log.New("stars.api")
	api := &API{
		starService:      starService,
		dashboardService: dashboardService,
		logger:           starLogger,
	}
	return api
}

func (api *API) getDashboardHelper(ctx context.Context, orgID int64, id int64, uid string) (*dashboards.Dashboard, response.Response) {
	var query dashboards.GetDashboardQuery

	if len(uid) > 0 {
		query = dashboards.GetDashboardQuery{UID: uid, ID: id, OrgID: orgID}
	} else {
		query = dashboards.GetDashboardQuery{ID: id, OrgID: orgID}
	}

	result, err := api.dashboardService.GetDashboard(ctx, &query)
	if err != nil {
		return nil, response.Error(http.StatusNotFound, "Dashboard not found", err)
	}

	return result, nil
}

func (api *API) GetStars(c *contextmodel.ReqContext) response.Response {
	query := star.GetUserStarsQuery{
		UserID: c.UserID,
	}

	iuserstars, err := api.starService.GetByUser(c.Req.Context(), &query)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to get user stars", err)
	}

	uids := []string{}
	for uid := range iuserstars.UserStars {
		uids = append(uids, uid)
	}

	return response.JSON(http.StatusOK, uids)
}

// swagger:route POST /user/stars/dashboard/{dashboard_id} signed_in_user starDashboard
//
// Star a dashboard.
//
// Stars the given Dashboard for the actual user.
//
// Deprecated: true
//
// Responses:
// 200: okResponse
// 400: badRequestError
// 401: unauthorisedError
// 403: forbiddenError
// 500: internalServerError
func (api *API) StarDashboard(c *contextmodel.ReqContext) response.Response {
	userID, err := identity.UserIdentifier(c.GetID())
	if err != nil {
		return response.Error(http.StatusBadRequest, "Only users and service accounts can star dashboards", nil)
	}

	id, err := strconv.ParseInt(web.Params(c.Req)[":id"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Invalid dashboard ID", nil)
	}

	api.logger.Warn("POST /user/stars/dashboard/{dashboard_id} is deprecated, please use POST /user/stars/dashboard/uid/{dashboard_uid} instead")

	cmd := star.StarDashboardCommand{UserID: userID, DashboardID: id, Updated: time.Now()}
	// nolint:staticcheck
	if cmd.DashboardID <= 0 {
		return response.Error(http.StatusBadRequest, "Missing dashboard id", nil)
	}

	if err := api.starService.Add(c.Req.Context(), &cmd); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to star dashboard", err)
	}

	return response.Success("Dashboard starred!")
}

// swagger:route POST /user/stars/dashboard/uid/{dashboard_uid} signed_in_user starDashboardByUID
//
// Star a dashboard.
//
// Stars the given Dashboard for the actual user.
//
// Responses:
// 200: okResponse
// 400: badRequestError
// 401: unauthorisedError
// 403: forbiddenError
// 500: internalServerError
func (api *API) StarDashboardByUID(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" {
		return response.Error(http.StatusBadRequest, "Invalid dashboard UID", nil)
	}

	userID, err := identity.UserIdentifier(c.GetID())
	if err != nil {
		return response.Error(http.StatusBadRequest, "Only users and service accounts can star dashboards", nil)
	}

	dash, rsp := api.getDashboardHelper(c.Req.Context(), c.GetOrgID(), 0, uid)
	if rsp != nil {
		return rsp
	}

	cmd := star.StarDashboardCommand{UserID: userID, DashboardID: dash.ID, DashboardUID: uid, OrgID: c.GetOrgID(), Updated: time.Now()}

	if err := api.starService.Add(c.Req.Context(), &cmd); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to star dashboard", err)
	}

	return response.Success("Dashboard starred!")
}

// swagger:route DELETE /user/stars/dashboard/{dashboard_id} signed_in_user unstarDashboard
//
// Unstar a dashboard.
//
// Deletes the starring of the given Dashboard for the actual user.
//
// Deprecated: true
//
// Please refer to the [new](#/signed_in_user/unstarDashboardByUID) API instead
//
// Responses:
// 200: okResponse
// 400: badRequestError
// 401: unauthorisedError
// 403: forbiddenError
// 500: internalServerError
func (api *API) UnstarDashboard(c *contextmodel.ReqContext) response.Response {
	id, err := strconv.ParseInt(web.Params(c.Req)[":id"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Invalid dashboard ID", nil)
	}

	userID, err := identity.UserIdentifier(c.GetID())
	if err != nil {
		return response.Error(http.StatusBadRequest, "Only users and service accounts can star dashboards", nil)
	}

	api.logger.Warn("DELETE /user/stars/dashboard/{dashboard_id} is deprecated, please use DELETE /user/stars/dashboard/uid/{dashboard_uid} instead")

	cmd := star.UnstarDashboardCommand{UserID: userID, DashboardID: id}
	// nolint:staticcheck
	if cmd.DashboardID <= 0 {
		return response.Error(http.StatusBadRequest, "Missing dashboard id", nil)
	}

	if err := api.starService.Delete(c.Req.Context(), &cmd); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to unstar dashboard", err)
	}

	return response.Success("Dashboard unstarred")
}

// swagger:route DELETE /user/stars/dashboard/uid/{dashboard_uid} signed_in_user unstarDashboardByUID
//
// Unstar a dashboard.
//
// Deletes the starring of the given Dashboard for the actual user.
//
// Responses:
// 200: okResponse
// 400: badRequestError
// 401: unauthorisedError
// 403: forbiddenError
// 500: internalServerError
func (api *API) UnstarDashboardByUID(c *contextmodel.ReqContext) response.Response {
	uid := web.Params(c.Req)[":uid"]
	if uid == "" {
		return response.Error(http.StatusBadRequest, "Invalid dashboard UID", nil)
	}

	userID, err := identity.UserIdentifier(c.GetID())
	if err != nil {
		return response.Error(http.StatusBadRequest, "Only users and service accounts can star dashboards", nil)
	}

	cmd := star.UnstarDashboardCommand{UserID: userID, DashboardUID: uid, OrgID: c.GetOrgID()}

	if err := api.starService.Delete(c.Req.Context(), &cmd); err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to unstar dashboard", err)
	}

	return response.Success("Dashboard unstarred")
}

// swagger:parameters starDashboard
type StarDashboardParams struct {
	// in:path
	// required:true
	DashboardID string `json:"dashboard_id"`
}

// swagger:parameters starDashboardByUID
type StarDashboardByUIDParams struct {
	// in:path
	// required:true
	DashboardUID string `json:"dashboard_uid"`
}

// swagger:parameters unstarDashboard
type UnstarDashboardParams struct {
	// in:path
	// required:true
	DashboardID string `json:"dashboard_id"`
}

// swagger:parameters unstarDashboardByUID
type UnstarDashboardByUIDParams struct {
	// in:path
	// required:true
	DashboardUID string `json:"dashboard_uid"`
}
