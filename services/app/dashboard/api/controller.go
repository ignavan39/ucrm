package api

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"

	"ucrm/app/auth"
	"ucrm/app/dashboard"
	"ucrm/app/models"

	"github.com/go-chi/chi"

	dashboardSettings "ucrm/app/dashboard-settings"
	"ucrm/pkg/httpext"
	"ucrm/pkg/logger"
)

type Controller struct {
	repo        dashboard.Repository
	webhookRepo dashboardSettings.CardWebhookRepository
}

func NewController(repo dashboard.Repository, webhookRepo dashboardSettings.CardWebhookRepository) *Controller {
	return &Controller{
		repo:        repo,
		webhookRepo: webhookRepo,
	}
}

// CreateOne godoc
// @Summary   Create dashboard
// @Tags      dashboards
// @Accept    json
// @Produce   json
// @Param     payload  body      CreateOnePayload  true  " "
// @Success   200      {object}  models.Dashboard
// @Failure   400      {object}  httpext.CommonError
// @Failure   401      {object}  httpext.CommonError
// @Failure   500      {object}  httpext.CommonError
// @Router    /dashboards/create [patch]
// @security  JWT
func (c *Controller) CreateOne(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload CreateDashboardPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	userId := auth.GetUserIdFromContext(ctx)
	dashboard, err := c.repo.Create(payload.Name, userId)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	httpext.JSON(w, *dashboard, http.StatusCreated)
}

// AddAccess godoc
// @Summary   Add Access
// @Accept    json
// @Tags      dashboards
// @Param     payload  body  AddAccessPayload  true  " "
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards/admin/addAccess [post]
// @security  JWT
func (c *Controller) AddAccess(w http.ResponseWriter, r *http.Request) {
	var payload AddAccessPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = payload.Validate()
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	dashboard, err := c.repo.GetOneInternal(payload.DashboardId)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if dashboard == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "dashboard not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	currentUser := auth.GetUserIdFromContext(r.Context())
	if payload.UserId == currentUser {
		httpext.JSON(w, httpext.CommonError{
			Error: "you can't changed your access",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = c.repo.AddAccess(payload.DashboardId, payload.UserId, payload.Access)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetOneDashboard godoc
// @Summary   Get dashboard
// @Tags      dashboards
// @Produce   json
// @Success   200  {object}  GetOneDashboardResponse
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards/{dashboardId} [get]
// @security  JWT
func (c *Controller) GetOneDashboard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dashboardId")
	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	dashboard, err := c.repo.GetOne(id)
	if err != nil {
		logger.Logger.Error("[dashboards/getOnde] ERROR :%s", err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if dashboard == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "dashboard not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	cardFields := make([]models.Field, 0)
	contactFields := make([]models.Field, 0)
	for _, field := range dashboard.Fields {
		if field.Type == FIELD_TYPE_CARD {
			cardFields = append(cardFields, field)
		} else if field.Type == FIELD_TYPE_CONTACT {
			contactFields = append(contactFields, field)
		}
	}

	response := GetOneDashboardResponse{
		dashboard.Id,
		dashboard.UpdatedAt,
		dashboard.Name,
		dashboard.AuthorId,
		dashboard.Pipelines,
		GetOneDashboardFields{
			cardFields,
			contactFields,
		},
	}

	httpext.JSON(w, &response, http.StatusOK)
}

// UpdateName    godoc
// @Summary   Update dashboard name
// @Tags      dashboards
// @Accept    json
// @Param     dashboardId  query  string             true  " "
// @Param     payload      body   UpdateNamePayload  true  " "
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards/admin/{dashboardId} [patch]
// @security  JWT
func (c *Controller) UpdateName(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dashboardId")
	var payload UpdateNamePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if len(payload.Name) < 2 {
		httpext.JSON(w, httpext.CommonError{
			Error: "name too short",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = c.repo.UpdateName(id, payload.Name)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteById godoc
// @Summary   Delete dashboard
// @Param     dashboardId  query  string  true  " "
// @Tags      dashboards
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards/admin/{dashboardId} [delete]
// @security  JWT
func (c *Controller) DeleteById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dashboardId")

	err := c.repo.DeleteById(id)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// AddWebhook    godoc
// @Summary   Add webhook
// @Tags      contacts
// @Accept    json
// @Param     dashboardId  query  string             true  " "
// @Param     payload      body   UpdateNamePayload  true  " "
// @Success   201
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards/admin/{dashboardId}/webhook [patch]
// @security  JWT
func (c *Controller) AddWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dashboardId")
	var payload AddWebhookPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if len(payload.Url) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "url to short",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = c.webhookRepo.AddCardWebhook(id, payload.Url, payload.Name)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// AddWebhook    godoc
// @Summary   Add webhook
// @Tags      dashboards
// @Accept    json
// @Produce   json
// @Param     dashboardId  query     string              true  " "
// @Param     payload      body      AddSettingsPayload  true  " "
// @Success   200          {object}  models.DashboardSettings
// @Failure   400          {object}  httpext.CommonError
// @Failure   401          {object}  httpext.CommonError
// @Failure   500          {object}  httpext.CommonError
// @Router    /dashboards/admin/{dashboardId}/settings [patch]
// @security  JWT
func (c *Controller) AddSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dashboardId")
	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	var payload AddSettingsPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	pwd := sha1.New()
	pwd.Write([]byte(payload.Secret))

	xClientToken := fmt.Sprintf("%x", pwd.Sum(nil))
	settings, err := c.repo.AddSettings(id, payload.Secret, xClientToken)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	httpext.JSON(w, settings, http.StatusOK)
}

// CreateCustomField    godoc
// @Summary   Create custom field
// @Tags      dashboards
// @Accept    json
// @Produce   json
// @Param     dashboardId  query     string          true  " "
// @Param     payload      body      AddCustomField  true  " "
// @Success   200          {object}  models.Field
// @Failure   400          {object}  httpext.CommonError
// @Failure   401          {object}  httpext.CommonError
// @Failure   500          {object}  httpext.CommonError
// @Router    /dashboards/{dashboardId}/custom-field [post]
// @security  JWT
func (c *Controller) CreateCustomField(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dashboardId := chi.URLParam(r, "dashboardId")

	if len(dashboardId) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing dashboardId: dashboards/createCustomField",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	var payload AddCustomField
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload: dashboards/createCustomField",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = payload.Validate()
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	field, err := c.repo.AddCustomField(dashboardId, payload.Name, payload.IsNullable, payload.FieldType)
	if err != nil {
		logger.Logger.Errorf("[dashboards/createCustomFields] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[CreateCustomField]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	httpext.JSON(w, field, http.StatusOK)
}

// DeleteCustomField    godoc
// @Summary   Delete custom field
// @Tags      dashboards
// @Param     fieldId      query     string          true  " "
// @Success   200
// @Failure   400          {object}  httpext.CommonError
// @Failure   401          {object}  httpext.CommonError
// @Failure   500          {object}  httpext.CommonError
// @Router    /dashboards/admin/custom-field/{fieldId} [delete]
// @security  JWT
func (c *Controller) DeleteCustomField(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fieldId := chi.URLParam(r, "fieldId")

	if len(fieldId) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing fieldId: dashboards/deleteCustomField",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err := c.repo.DeleteCustomField(fieldId)
	if err != nil {
		logger.Logger.Errorf("[dashboards/deleteCustomFields] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[DeleteCustomField]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RemoveAccess    godoc
// @Summary   Remove access
// @Tags      dashboards
// @Accept    json
// @Produce   json
// @Param     dashboardId  query     string  true  " "
// @Param     userId       query     string  true  " "
// @Success   200          {object}  string
// @Failure   400          {object}  httpext.CommonError
// @Failure   401          {object}  httpext.CommonError
// @Failure   500          {object}  httpext.CommonError
// @Router    /dashboards/admin/removeAccess/{dashboardId}/{userId} [delete]
// @security  JWT
func (c *Controller) RemoveAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "dashboardId")
	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	userId := chi.URLParam(r, "userId")
	if len(userId) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong user id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	currentUser := auth.GetUserIdFromContext(r.Context())
	if userId == currentUser {
		httpext.JSON(w, httpext.CommonError{
			Error: "you can't changed your access",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err := c.repo.RemoveAccess(id, userId)
	if err != nil {
		logger.Logger.Errorf("[dashboards/RemoveAccess] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[RemoveAccess]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateAccess    godoc
// @Summary   Update access
// @Tags      dashboards
// @Accept    json
// @Produce   json
// @Param     AddAccessPayload  body  string  true  " "
// @Success   200
// @Failure   400  {string}  string  "[CreateOne]:  {error}"
// @Failure   400  {object}  httpext.CommonError
// @Router    /dashboards/admin/updateAccess [patch]
// @security  JWT
func (c *Controller) UpdateAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload AddAccessPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = payload.Validate()
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	currentUser := auth.GetUserIdFromContext(r.Context())
	if payload.UserId == currentUser {
		httpext.JSON(w, httpext.CommonError{
			Error: "you can't changed your access",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = c.repo.UpdateAccess(payload.DashboardId, payload.UserId, payload.Access)
	if err != nil {
		logger.Logger.Errorf("[dashboards/UpdateAccess] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[UpdateAccess]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetByUser    godoc
// @Summary   Get user`s dashboards
// @Tags      dashboards
// @Produce   json
// @Success   200  {object}  []models.Dashboard
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /dashboards [get]
// @security  JWT
func (c *Controller) GetByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userId := auth.GetUserIdFromContext(ctx)

	dashboards, err := c.repo.GetOneByUser(userId)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	httpext.JSON(w, dashboards, http.StatusOK)
}
