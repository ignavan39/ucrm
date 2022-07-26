package pg

import (
	"context"
	"database/sql"
	"errors"

	"ucrm/app/dashboard/api"
	"ucrm/app/models"

	sq "github.com/Masterminds/squirrel"

	repository "ucrm/app/contact"
	"ucrm/pkg/logger"
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

func (r *Repository) GetOne(ctx context.Context, contactId string) (*models.Contact, error) {
	contact := &models.Contact{}

	rows, err := sq.Select("c.id", "c.dashboard_id", "c.card_id", "c.name",
		"c.phone", "c.city", "f.name", "cf.*").
		From("contacts c").
		Where(sq.Eq{"c.id": contactId}).
		LeftJoin("card_fields cf on cf.card_id = c.id").
		LeftJoin("fields f on f.id = cf.field_id").
		RunWith(r.pool.Read()).
		PlaceholderFormat(sq.Dollar).
		Query()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		logger.Logger.Errorf("[card/GetOneCard] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		return nil, err
	}

	defer rows.Close()
	fields := []models.ContactField{}

	for rows.Next() {
		var field models.ContactField
		var name, id, contactId, fieldId, value sql.NullString

		if err := rows.Scan(&contact.Id, &contact.DashboardId, &contact.CardId, &contact.Name, &contact.Phone, &contact.City,
			&name, &id, &contactId, &fieldId, &value,
		); err != nil {
			logger.Logger.Errorf("[card/GetOneCard] CTX: [%v], ERROR:[%s]", ctx, err.Error())
			return nil, err
		}

		if name.Valid {
			field.Name = name.String
			field.Id = id.String
			field.ContactId = contactId.String
			field.FieldId = fieldId.String
			field.Value = &value.String

			fields = append(fields, field)
		}

	}
	contact.Fields = fields

	return contact, nil
}

func (r *Repository) Create(ctx context.Context, dashboardId string, cardId *string, name string, phone string, city string, fields *map[string]string) (*models.Contact, error) {
	contact := &models.Contact{}
	isUpdateWithTransaction := fields != nil
	var tx *sql.Tx

	if isUpdateWithTransaction {
		var err error
		tx, err = r.pool.Write().Begin()
		if err != nil {
			return nil, err
		}
	}

	qb := sq.Insert("contacts").
		Columns("dashboard_id", "card_id", "name", "phone", "city").
		Suffix("returning id, dashboard_id, card_id, name, phone, city").
		Values(dashboardId, &cardId, name, phone, city)

	if isUpdateWithTransaction {
		qb = qb.RunWith(tx)
	} else {
		qb = qb.RunWith(r.pool.Write())
	}
	row := qb.PlaceholderFormat(sq.Dollar).
		QueryRow()
	if err := row.Scan(&contact.Id, &contact.DashboardId, &contact.CardId, &contact.Name, &contact.Phone, &contact.City); err != nil {
		logger.Logger.Errorf("[contact/Delete] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		return nil, err
	}

	if fields != nil {
		fieldsRow, err := sq.Select("id").
			From("fields f").
			Where(sq.Eq{"f.dashboard_id": contact.DashboardId, "f.type": api.FIELD_TYPE_CONTACT}).
			RunWith(r.pool.Read()).
			PlaceholderFormat(sq.Dollar).
			Query()

		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		} else {
			fieldsIds := make([]string, 0)

			for fieldsRow.Next() {
				var fieldId string
				if err = fieldsRow.Scan(&fieldId); err != nil {
					return nil, err
				}

				fieldsIds = append(fieldsIds, fieldId)
			}

			fieldsRow.Close()

			noValueFieldIds := make([]string, 0)
			if fields != nil {
				fieldsMap := *fields

				for _, id := range fieldsIds {
					value, found := fieldsMap[id]
					if !found {
						noValueFieldIds = append(noValueFieldIds, id)
					} else {
						_, err := sq.Insert("contact_fields").
							Columns("contact_id", "field_id", "value").
							Values(contact.Id, id, value).
							RunWith(tx).
							PlaceholderFormat(sq.Dollar).
							Exec()

						if err != nil {
							if err = tx.Rollback(); err != nil {
								return nil, err
							}
							return nil, repository.ErrFieldNotFound
						}
					}
				}
			} else {
				noValueFieldIds = fieldsIds
			}

			for _, id := range noValueFieldIds {
				_, err := sq.Insert("contact_fields").
					Columns("contact_id", "field_id").
					Values(contact.Id, id).
					RunWith(tx).
					PlaceholderFormat(sq.Dollar).
					Exec()

				if err != nil {
					if err = tx.Rollback(); err != nil {
						return nil, err
					}
					return nil, err
				}
			}
		}
	}

	return contact, nil
}

func (r *Repository) Update(ctx context.Context, contactId string, name *string, phone *string, city *string, contactFields *map[string]string) error {
	var isContactDataUpdate bool = name != nil || phone != nil || city != nil
	var isUpdateWithTransaction bool = isContactDataUpdate && contactFields != nil
	var tx *sql.Tx

	if contactFields != nil {
		if isUpdateWithTransaction {
			var err error
			tx, err = r.pool.Write().Begin()
			if err != nil {
				logger.Logger.Errorf("[card/update] CTX: [%v], ERROR:[%s]", ctx, err.Error())
				return err
			}
		}

		for key, value := range *contactFields {
			qb := sq.Update("contact_fields c").
				Set("value", value).
				Where(sq.Eq{"c.id": key})

			if isUpdateWithTransaction {
				qb = qb.RunWith(tx)
			} else {
				qb = qb.RunWith(r.pool.Write())
			}

			_, err := qb.
				PlaceholderFormat(sq.Dollar).
				Exec()
			if err != nil {
				if isUpdateWithTransaction {
					if err := tx.Rollback(); err != nil {
						logger.Logger.Errorf("[card/update] CTX: [%v], ERROR:[%s]", ctx, err.Error())
						return err
					}
				}

				if errors.Is(err, sql.ErrNoRows) {
					return nil
				}
				return err
			}
		}
	}

	if isContactDataUpdate {
		qb := sq.Update("contacts")
		if name != nil {
			qb = qb.Set("name", name)
		}

		if phone != nil {
			qb = qb.Set("phone", phone)
		}

		if city != nil {
			qb = qb.Set("city", city)
		}

		qb = qb.Where(sq.Eq{"id": contactId}).
			PlaceholderFormat(sq.Dollar)

		if isUpdateWithTransaction {
			qb = qb.RunWith(tx)
		} else {
			qb = qb.RunWith(r.pool.Write())
		}

		_, err := qb.Exec()
		if err != nil {
			if isUpdateWithTransaction {
				if err := tx.Rollback(); err != nil {
					logger.Logger.Errorf("[card/update] CTX: [%v], ERROR:[%s]", ctx, err.Error())
					return err
				}
			}

			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}

		if isUpdateWithTransaction {
			if err = tx.Commit(); err != nil {
				logger.Logger.Errorf("[card/update] CTX: [%v], ERROR:[%s]", ctx, err.Error())
				return err
			}
		}
	}

	return nil
}

func (r *Repository) Rename(ctx context.Context, contactId string, newName string) error {
	_, err := sq.Update("contacts c").
		Set("name", newName).
		Where(sq.Eq{"c.id": contactId}).
		RunWith(r.pool.Write()).
		PlaceholderFormat(sq.Dollar).
		Exec()
	if err != nil {
		logger.Logger.Errorf("[contact/Delete] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		return err
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, contactId string) error {
	_, err := sq.Delete("contacts").
		Where(sq.Eq{"id": contactId}).
		RunWith(r.pool.Write()).
		PlaceholderFormat(sq.Dollar).
		Exec()
	if err != nil {
		logger.Logger.Errorf("[contact/Delete] CTX: [%v], ERROR:[%s]", ctx, err.Error())
		return err
	}

	return nil
}
