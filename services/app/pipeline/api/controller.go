package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"ucrm/app/models"
	"ucrm/app/pipeline"
	"ucrm/pkg/httpext"
	"ucrm/pkg/logger"

	"github.com/go-chi/chi"
)

type Controller struct {
	repo pipeline.Repository
}

func NewController(repo pipeline.Repository) *Controller {
	return &Controller{repo: repo}
}

// CreateOne  godoc
// @Summary   Create pipeline
// @Tags      pipelines
// @Accept    json
// @Produce   json
// @Param     payload  body  CreateOnePayload  true  " "
// @Success   201  {object}  CreatePipelineResponse
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /pipelines/create [post]
// @security  JWT
func (c *Controller) CreateOne(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload CreateOnePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload: pipelines/createOne",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	pipeline, err := c.repo.Create(payload.Name, payload.DashboardId)
	if err != nil {
		logger.Logger.Errorf("[pipeline/create] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	var Response CreatePipelineResponse
	Response.Pipeline = *pipeline

	httpext.JSON(w, Response, http.StatusCreated)
}

// UpdateName godoc
// @Summary   Update dashboard`s name
// @Tags      pipelines
// @Accept    json
// @Param     payload     body   UpdateDashboardNamePayload  true  " "
// @Param     pipelineId  query  string                      true  " "
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /pipelines/{pipelineId} [patch]
// @security  JWT
func (c *Controller) UpdateName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload UpdateDashboardNamePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "failed decode payload: pipelines/updateName",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "pipelineId")
	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing id: pipelines/updateName",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err = c.repo.UpdateName(id, payload.Name)
	if err != nil {
		logger.Logger.Errorf("[pipeline/updateName] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteById godoc
// @Summary   Delete pipeline
// @Tags      pipelines
// @Param     pipelineId  query  string  true  " "
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /pipelines/{pipelineId} [delete]
// @security  JWT
func (c *Controller) DeleteById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "pipelineId")

	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing id: pipelines/deleteById",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err := c.repo.DeleteById(id)
	if err != nil {
		logger.Logger.Errorf("[pipeline/delete] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

// UpdateOrder godoc
// @Summary   Update pipeline order
// @Tags      pipelines
// @Accept    json
// @Param     pipelineId  query  string  true  " "
// @Param     order       query  string  true  " "
// @Success   200
// @Failure   400  {object}  httpext.CommonError
// @Failure   401  {object}  httpext.CommonError
// @Failure   500  {object}  httpext.CommonError
// @Router    /pipelines/order/{pipelineId}/{order} [patch]
// @security  JWT
func (c *Controller) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pipelineId := chi.URLParam(r, "pipelineId")
	orderQuery := chi.URLParam(r, "order")

	if len(pipelineId) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing id: pipelines/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if len(orderQuery) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing order: pipelines/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	newOrder, err := strconv.Atoi(orderQuery)
	if err != nil || newOrder < 1 {
		httpext.JSON(w, httpext.CommonError{
			Error: "incorrect value for order: pipelines/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	pipelines, err := c.repo.GetAllByPipeline(pipelineId)
	if err != nil {
		logger.Logger.Errorf("[pipeline/updateOrder] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if len(pipelines) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "pipelines is empty",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	maxOrder := 0
	var pipeline models.Pipeline
	for _, p := range pipelines {
		if p.Id == pipelineId {
			if p.Order == newOrder {
				httpext.JSON(w, httpext.CommonError{
					Error: "Incorrect new order for update",
					Code:  http.StatusBadRequest,
				}, http.StatusBadRequest)
				return
			}
			pipeline = p
		}
		if p.Order >= maxOrder {
			maxOrder = p.Order
		}
	}

	if newOrder > maxOrder {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong order",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	pipelineIdToNewOrder := make(map[string]int)

	for _, p := range pipelines {
		if newOrder > pipeline.Order {
			if p.Id == pipelineId {
				if p.Order == newOrder {
					continue
				} else {
					pipelineIdToNewOrder[p.Id] = newOrder
				}
			} else if p.Order <= newOrder && p.Order > pipeline.Order {
				pipelineIdToNewOrder[p.Id] = p.Order - 1
			}
		} else {
			if p.Id == pipelineId {
				if p.Order == newOrder {
					continue
				} else {
					pipelineIdToNewOrder[p.Id] = newOrder
				}
			} else if p.Order >= newOrder && p.Order < pipeline.Order {
				pipelineIdToNewOrder[p.Id] = p.Order + 1
			}
		}
	}

	err = c.repo.UpdateOrders(pipelineIdToNewOrder)
	if err != nil {
		logger.Logger.Errorf("[pipeline/updateOrder] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[UpdateOrder]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
