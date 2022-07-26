package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"ucrm/app/card"
	repository "ucrm/app/card"

	"github.com/go-chi/chi"

	dashboardSettings "ucrm/app/dashboard-settings"
	"ucrm/app/models"
	"ucrm/pkg/httpext"
	"ucrm/pkg/logger"
)

type Controller struct {
	repo            card.Repository
	cardWebhookRepo dashboardSettings.CardWebhookRepository
}

func NewController(repo card.Repository, cardWebhookRepo dashboardSettings.CardWebhookRepository) *Controller {
	return &Controller{
		repo:            repo,
		cardWebhookRepo: cardWebhookRepo,
	}
}

// CreateOne godoc
// @Summary   Create card
// @Tags      cards
// @Accept    json
// @Produce   json
// @Param     payload  body      CreateOnePayload  true  " "
// @Success   201      {object}  models.Card
// @Failure   400      {object}  httpext.CommonError
// @Failure   401      {object}  httpext.CommonError
// @Failure   500      {object}  httpext.CommonError
// @Router    /cards/create [post]
// @security  JWT
func (c *Controller) CreateOne(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload CreateOnePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "[CreateOne] failed decode payload",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	res, err := c.repo.CreateOne(payload.Name, payload.PipelineId, payload.Fields)
	if err != nil {
		if errors.Is(err, repository.ErrFieldNotFound) {
			httpext.JSON(w, httpext.CommonError{
				Error: fmt.Sprintf("[CreateOne]:%s", err.Error()),
				Code:  http.StatusBadRequest,
			}, http.StatusBadRequest)
			return
		}

		logger.Logger.Errorf("[card/createOne] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[CreateOne]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	webhook, err := c.cardWebhookRepo.GetCardWebhookByPipelineId(payload.PipelineId)
	if err != nil {
		logger.Logger.Errorf("[card/createOne] CTX: [%v], ERROR:[%s]", err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: "[CreateOne] failed to get webhook",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if webhook != nil {
		go card.SendCardUpdatesToSubscriber(webhook.Url, res, nil)
	}

	httpext.JSON(w, res, http.StatusCreated)
}

// Delete godoc
// @Summary   Delete card
// @Tags      cards
// @Produce   json
// @Param     cardId  query     string  true  " "
// @Success   200     {object}  models.Card
// @Failure   400     {object}  httpext.CommonError
// @Failure   401     {object}  httpext.CommonError
// @Failure   500     {object}  httpext.CommonError
// @Router    /cards/{cardId} [delete]
// @security  JWT
func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "cardId")

	res, err := c.repo.GetOneWithoutRelations(id)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[Delete]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if res == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "card not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	err = c.repo.Delete(id)
	if err != nil {
		logger.Logger.Errorf("[card/delete] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[Delete]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	webhook, err := c.cardWebhookRepo.GetCardWebhookByPipelineId(res.PipelineId)
	if err != nil {
		logger.Logger.Errorf("[card/delete] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: "[Delete] failed to get webhook",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if webhook != nil {
		go card.SendCardUpdatesToSubscriber(webhook.Url, nil, res)
	}

	httpext.JSON(w, res, http.StatusOK)
}

// Update godoc
// @Summary  Update card
// @Description
// @Accept    json
// @Produce   json
// @Tags      cards
// @Param     cardId   query     string            true  " "
// @Param     payload  body      UpdateOnePayload  true  " "
// @Success   200      {object}  models.Card
// @Failure   400      {object}  httpext.CommonError
// @Failure   401      {object}  httpext.CommonError
// @Failure   500      {object}  httpext.CommonError
// @Router    /cards [patch]
// @security  JWT
func (c *Controller) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload UpdateOnePayload
	id := chi.URLParam(r, "cardId")

	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "[Update] wrong id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "[Update] failed decode payload",
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

	oldCard, err := c.repo.GetOne(id)
	if err != nil {
		logger.Logger.Errorf("[card/update] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[Update]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if oldCard == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "card not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	updatedCard, err := c.repo.Update(id, payload.Name, payload.Fields)
	if err != nil {
		logger.Logger.Errorf("[card/update] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[Update]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if updatedCard == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "card not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	webhook, err := c.cardWebhookRepo.GetCardWebhookByPipelineId(oldCard.PipelineId)
	if err != nil {
		logger.Logger.Errorf("[card/update] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: "[Update] failed to get pipeline",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if webhook != nil {
		go card.SendCardUpdatesToSubscriber(webhook.Url, oldCard, updatedCard)
	}

	httpext.JSON(w, updatedCard, http.StatusOK)
}

// GetOne godoc
// @Summary      Get one card
// @Description  Get one card by id
// @Tags         cards
// @Accept       json
// @Produce      json
// @Param        cardId  query     string  true      " "
// @Success      200     {object}  models.Card
// @Failure      400     {object}  httpext.CommonError
// @Failure      401     {object}  httpext.CommonError
// @Failure      500     {object}  httpext.CommonError
// @Router       /cards/{cardId} [get]
// @security     JWT
func (c *Controller) GetOne(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "cardId")

	if len(id) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "[GetOne] wrong id",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	card, err := c.repo.GetOne(id)
	if err != nil {
		logger.Logger.Errorf("[card/getOne] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[GetOne]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	if card == nil {
		httpext.JSON(w, httpext.CommonError{
			Error: "card not found",
			Code:  http.StatusNotFound,
		}, http.StatusNotFound)
		return
	}

	httpext.JSON(w, card, http.StatusOK)
}

// UpdateOrder godoc
// @Summary      Update order
// @Description  Update order
// @Tags         cards
// @Param        cardId      query  string  true  " "
// @Param        pipelineId  query  string  true  " "
// @Param        order       query  string  true  " "
// @Success      200
// @Failure      400  {object}  httpext.CommonError
// @Failure      401  {object}  httpext.CommonError
// @Failure      500  {object}  httpext.CommonError
// @Router       /cards/order/{pipelineId}/{cardId}/order [get]
// @security     JWT
func (cr *Controller) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cardId := chi.URLParam(r, "cardId")
	orderQuery := chi.URLParam(r, "order")

	if len(cardId) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing cardId: cards/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	if len(orderQuery) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing order: cards/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	newOrder, err := strconv.Atoi(orderQuery)
	if err != nil || newOrder < 1 {
		httpext.JSON(w, httpext.CommonError{
			Error: "missing order: cards/updateOrder",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	cards, err := cr.repo.GetAllByPipelineId(cardId)
	if err != nil {
		logger.Logger.Errorf("[card/updateOrder] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: err.Error(),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	var card models.Card
	maxOrder := 0

	if len(cards) == 0 {
		httpext.JSON(w, httpext.CommonError{
			Error: "cards is empty",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	for _, c := range cards {
		if c.Id == cardId {
			if c.Order == newOrder {
				httpext.JSON(w, httpext.CommonError{
					Error: "Incorrect new order for update",
					Code:  http.StatusBadRequest,
				}, http.StatusBadRequest)
				return
			}
			card = c
		}
		if c.Order >= maxOrder {
			maxOrder = c.Order
		}
	}

	if newOrder > maxOrder {
		httpext.JSON(w, httpext.CommonError{
			Error: "wrong order",
			Code:  http.StatusBadRequest,
		}, http.StatusBadRequest)
		return
	}

	cardIdToNewOrder := make(map[string]int)

	for _, c := range cards {
		if newOrder > card.Order {
			if c.Id == cardId {
				if c.Order == newOrder {
					continue
				} else {
					cardIdToNewOrder[c.Id] = newOrder
				}
			} else if c.Order <= newOrder && c.Order > card.Order {
				cardIdToNewOrder[c.Id] = c.Order - 1
			}
		} else {
			if c.Id == cardId {
				if c.Order == newOrder {
					continue
				} else {
					cardIdToNewOrder[c.Id] = newOrder
				}
			} else if c.Order >= newOrder && c.Order < card.Order {
				cardIdToNewOrder[c.Id] = c.Order + 1
			}
		}
	}

	err = cr.repo.UpdateOrders(cardIdToNewOrder)
	if err != nil {
		logger.Logger.Errorf("[card/updateOrder] ctx: %v, error: %s", ctx, err.Error())
		httpext.JSON(w, httpext.CommonError{
			Error: fmt.Sprintf("[UpdateOrder]:%s", err.Error()),
			Code:  http.StatusInternalServerError,
		}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
