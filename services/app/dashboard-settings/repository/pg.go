package database

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"ucrm/app/models"
	"ucrm/pkg/pg"
)

type Repository struct {
	pool pg.Pool
}

func NewRepository(pool pg.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (s *Repository) AddCardWebhook(dashboardId string, url string, name *string) error {
	_, err := sq.Insert("card_webhook").
		Columns("url", "name", "dashboard_id").
		Values(url, name, dashboardId).
		Suffix("on conflict (dashboard_id) do update set url = ?", url).
		RunWith(s.pool.Write()).
		PlaceholderFormat(sq.Dollar).
		Exec()
	if err != nil {
		return err
	}

	return err
}

func (s *Repository) GetCardWebhookByDashboardId(dashboardId string) (*models.CardWebhook, error) {
	row := sq.Select("url", "name", "dashboard_id").
		From("card_webhook").
		RunWith(s.pool.Read()).
		PlaceholderFormat(sq.Dollar).
		QueryRow()

	var webhook models.CardWebhook
	if err := row.Scan(&webhook.Url, &webhook.Name, &webhook.DashboardId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &webhook, nil
}

func (s *Repository) GetCardWebhookByPipelineId(pipelineId string) (*models.CardWebhook, error) {
	row := sq.Select("c.url", "c.name", "c.dashboard_id").
		From("card_webhook c").
		LeftJoin("dashboards d on c.dashboard_id = d.id").
		LeftJoin("pipelines p on p.dashboard_id = d.id").
		Where(sq.Eq{"p.id": pipelineId}).
		RunWith(s.pool.Read()).
		PlaceholderFormat(sq.Dollar).
		QueryRow()

	var webhook models.CardWebhook
	if err := row.Scan(&webhook.Url, &webhook.Name, &webhook.DashboardId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &webhook, nil
}
