package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

type OrderService struct {
	orders      *repository.OrderRepo
	items       *repository.ItemRepo
	itemOptions *repository.ItemOptionRepo
	optionTypes *repository.OptionTypeRepo
	users       *repository.UserRepo
	tx          *repository.TxManager
}

func NewOrderService(
	orders *repository.OrderRepo,
	items *repository.ItemRepo,
	itemOptions *repository.ItemOptionRepo,
	optionTypes *repository.OptionTypeRepo,
	users *repository.UserRepo,
	tx *repository.TxManager,
) *OrderService {
	return &OrderService{orders: orders, items: items, itemOptions: itemOptions, optionTypes: optionTypes, users: users, tx: tx}
}

// Create — создание заказа с полной проверкой цен.
func (s *OrderService) Create(ctx context.Context, userID uuid.UUID, req model.OrderCreateRequest) (model.OrderDetailDTO, error) {
	if err := validateOrderReq(req); err != nil {
		return model.OrderDetailDTO{}, err
	}

	// Прогрев: проверяем все товары и опции ДО начала tx, чтобы быстро отказать.
	// Но окончательная запись — в tx.
	type computedItem struct {
		req         model.OrderItemCreate
		item        model.Item
		basePrice   int // с учётом sale
		options     []model.ItemOption
		optionTypes map[uuid.UUID]model.OptionType
		totalPrice  int
	}
	computed := make([]computedItem, 0, len(req.Items))
	totalPrice := 0

	typeIDSet := map[uuid.UUID]bool{}
	for _, line := range req.Items {
		it, err := s.items.GetByID(ctx, line.ItemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.OrderDetailDTO{}, model.NewInvalidInput(fmt.Sprintf("Товар %s не найден", line.ItemID))
			}
			return model.OrderDetailDTO{}, fmt.Errorf("get item: %w", err)
		}
		if it.Hidden {
			return model.OrderDetailDTO{}, model.NewConflict(fmt.Sprintf("Товар %s сейчас недоступен к заказу", it.ID))
		}
		base := it.FinalPrice()
		opts := make([]model.ItemOption, 0, len(line.OptionIDs))
		unitTotal := base
		for _, oid := range line.OptionIDs {
			o, err := s.itemOptions.GetByIDForItem(ctx, it.ID, oid)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return model.OrderDetailDTO{}, model.NewInvalidInput(fmt.Sprintf("Опция %s не принадлежит товару %s", oid, it.ID))
				}
				return model.OrderDetailDTO{}, fmt.Errorf("get option: %w", err)
			}
			opts = append(opts, o)
			unitTotal += o.Price
			typeIDSet[o.TypeID] = true
		}
		computed = append(computed, computedItem{req: line, item: it, basePrice: base, options: opts, totalPrice: unitTotal})
		totalPrice += unitTotal * line.Quantity
	}

	// Грузим все option types разом.
	typeIDs := make([]uuid.UUID, 0, len(typeIDSet))
	for t := range typeIDSet {
		typeIDs = append(typeIDs, t)
	}
	types, err := s.optionTypes.ListByIDs(ctx, typeIDs)
	if err != nil {
		return model.OrderDetailDTO{}, fmt.Errorf("option types: %w", err)
	}
	typesMap := make(map[uuid.UUID]model.OptionType, len(types))
	for _, t := range types {
		typesMap[t.ID] = t
	}

	var createdOrder model.Order
	err = s.tx.Run(ctx, func(ctx context.Context) error {
		orderID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("uuid: %w", err)
		}
		o := model.Order{
			ID:              orderID,
			UserID:          userID,
			Status:          model.OrderStatusCreated,
			TotalPrice:      totalPrice,
			CustomerComment: req.CustomerComment,
			ContactPhone:    strings.TrimSpace(req.ContactPhone),
			ContactFullName: strings.TrimSpace(req.ContactFullName),
		}
		if err := s.orders.CreateOrder(ctx, &o); err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		for _, c := range computed {
			oiID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("uuid: %w", err)
			}
			oi := model.OrderItem{
				ID:                  oiID,
				OrderID:             orderID,
				ItemID:              c.item.ID,
				Quantity:            c.req.Quantity,
				UnitBasePrice:       c.basePrice,
				UnitTotalPrice:      c.totalPrice,
				ItemNameSnapshot:    c.item.Name,
				ItemArticulSnapshot: c.item.Articul,
			}
			if err := s.orders.CreateOrderItem(ctx, &oi); err != nil {
				return fmt.Errorf("create order_item: %w", err)
			}
			for _, opt := range c.options {
				oioID, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("uuid: %w", err)
				}
				t := typesMap[opt.TypeID]
				oio := model.OrderItemOption{
					ID:                oioID,
					OrderItemID:       oiID,
					TypeCodeSnapshot:  t.Code,
					TypeLabelSnapshot: t.Label,
					ValueSnapshot:     opt.Value,
					PriceSnapshot:     opt.Price,
				}
				if err := s.orders.CreateOrderItemOption(ctx, &oio); err != nil {
					return fmt.Errorf("create order_item_option: %w", err)
				}
			}
		}
		createdOrder = o
		return nil
	})
	if err != nil {
		return model.OrderDetailDTO{}, err
	}

	return s.getMyDetail(ctx, createdOrder)
}

// ListMy — история заказов пользователя.
func (s *OrderService) ListMy(ctx context.Context, userID uuid.UUID, status *model.OrderStatus, p model.Pagination) (model.ListPage[model.OrderListItemDTO], error) {
	p.Clamp(20, 100)
	total, err := s.orders.CountForUser(ctx, userID, status)
	if err != nil {
		return model.ListPage[model.OrderListItemDTO]{}, fmt.Errorf("count: %w", err)
	}
	items, err := s.orders.ListForUser(ctx, userID, status, p)
	if err != nil {
		return model.ListPage[model.OrderListItemDTO]{}, fmt.Errorf("list: %w", err)
	}
	if items == nil {
		items = []model.OrderListItemDTO{}
	}
	return model.ListPage[model.OrderListItemDTO]{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
}

// GetMy — детали моего заказа с проверкой ownership.
func (s *OrderService) GetMy(ctx context.Context, userID, orderID uuid.UUID) (model.OrderDetailDTO, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.OrderDetailDTO{}, model.NewNotFound("Заказ не найден")
		}
		return model.OrderDetailDTO{}, fmt.Errorf("get order: %w", err)
	}
	if o.UserID != userID {
		// 404, чтобы не светить факт существования чужого заказа.
		return model.OrderDetailDTO{}, model.NewNotFound("Заказ не найден")
	}
	return s.getMyDetail(ctx, o)
}

func (s *OrderService) getMyDetail(ctx context.Context, o model.Order) (model.OrderDetailDTO, error) {
	items, options, err := s.loadOrderItems(ctx, o.ID)
	if err != nil {
		return model.OrderDetailDTO{}, err
	}
	return model.OrderDetailDTO{
		ID:              o.ID,
		Status:          o.Status,
		TotalPrice:      o.TotalPrice,
		CustomerComment: o.CustomerComment,
		ContactPhone:    o.ContactPhone,
		ContactFullName: o.ContactFullName,
		Items:           composeOrderItems(items, options),
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}, nil
}

// ---------- Admin ----------

func (s *OrderService) AdminList(ctx context.Context, f model.OrderAdminListFilter) (model.ListPage[model.OrderAdminListItemDTO], error) {
	f.Pagination.Clamp(20, 100)
	total, err := s.orders.CountAdmin(ctx, f)
	if err != nil {
		return model.ListPage[model.OrderAdminListItemDTO]{}, fmt.Errorf("count admin: %w", err)
	}
	items, err := s.orders.ListAdmin(ctx, f)
	if err != nil {
		return model.ListPage[model.OrderAdminListItemDTO]{}, fmt.Errorf("list admin: %w", err)
	}
	if items == nil {
		items = []model.OrderAdminListItemDTO{}
	}
	return model.ListPage[model.OrderAdminListItemDTO]{Items: items, Total: total, Limit: f.Pagination.Limit, Offset: f.Pagination.Offset}, nil
}

func (s *OrderService) AdminGet(ctx context.Context, orderID uuid.UUID) (model.OrderAdminDetailDTO, error) {
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.OrderAdminDetailDTO{}, model.NewNotFound("Заказ не найден")
		}
		return model.OrderAdminDetailDTO{}, fmt.Errorf("get order: %w", err)
	}
	return s.composeAdminDetail(ctx, o)
}

func (s *OrderService) AdminPatchStatus(ctx context.Context, orderID uuid.UUID, target model.OrderStatus) (model.OrderAdminDetailDTO, error) {
	if !isValidStatus(target) {
		return model.OrderAdminDetailDTO{}, model.NewInvalidInput("Некорректный статус")
	}
	o, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.OrderAdminDetailDTO{}, model.NewNotFound("Заказ не найден")
		}
		return model.OrderAdminDetailDTO{}, fmt.Errorf("get order: %w", err)
	}
	if !model.CanTransition(o.Status, target) {
		return model.OrderAdminDetailDTO{}, model.NewConflict(fmt.Sprintf("Недопустимый переход статуса: %s → %s", o.Status, target))
	}
	o, err = s.orders.UpdateStatus(ctx, orderID, target)
	if err != nil {
		return model.OrderAdminDetailDTO{}, fmt.Errorf("update status: %w", err)
	}
	return s.composeAdminDetail(ctx, o)
}

func (s *OrderService) AdminPatchNote(ctx context.Context, orderID uuid.UUID, note *string) (model.OrderAdminDetailDTO, error) {
	o, err := s.orders.UpdateAdminNote(ctx, orderID, note)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.OrderAdminDetailDTO{}, model.NewNotFound("Заказ не найден")
		}
		return model.OrderAdminDetailDTO{}, fmt.Errorf("update admin_note: %w", err)
	}
	return s.composeAdminDetail(ctx, o)
}

func (s *OrderService) composeAdminDetail(ctx context.Context, o model.Order) (model.OrderAdminDetailDTO, error) {
	u, err := s.users.GetByID(ctx, o.UserID)
	if err != nil {
		return model.OrderAdminDetailDTO{}, fmt.Errorf("get user: %w", err)
	}
	items, options, err := s.loadOrderItems(ctx, o.ID)
	if err != nil {
		return model.OrderAdminDetailDTO{}, err
	}
	return model.OrderAdminDetailDTO{
		ID: o.ID,
		User: model.UserFullShortDTO{
			ID: u.ID, Email: u.Email, FullName: u.FullName, Phone: u.Phone,
		},
		Status:          o.Status,
		TotalPrice:      o.TotalPrice,
		CustomerComment: o.CustomerComment,
		AdminNote:       o.AdminNote,
		ContactPhone:    o.ContactPhone,
		ContactFullName: o.ContactFullName,
		Items:           composeOrderItems(items, options),
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}, nil
}

// ---------- helpers ----------

func (s *OrderService) loadOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, map[uuid.UUID][]model.OrderItemOption, error) {
	items, err := s.orders.ListOrderItems(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("list order items: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	opts, err := s.orders.ListOrderItemOptions(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list options: %w", err)
	}
	return items, opts, nil
}

func composeOrderItems(items []model.OrderItem, opts map[uuid.UUID][]model.OrderItemOption) []model.OrderItemDTO {
	out := make([]model.OrderItemDTO, 0, len(items))
	for _, it := range items {
		dto := model.OrderItemDTO{
			ID:                  it.ID,
			ItemID:              it.ItemID,
			ItemNameSnapshot:    it.ItemNameSnapshot,
			ItemArticulSnapshot: it.ItemArticulSnapshot,
			Quantity:            it.Quantity,
			UnitBasePrice:       it.UnitBasePrice,
			UnitTotalPrice:      it.UnitTotalPrice,
			Options:             []model.OrderItemOptionDTO{},
		}
		for _, o := range opts[it.ID] {
			dto.Options = append(dto.Options, model.OrderItemOptionDTO{
				TypeCodeSnapshot:  o.TypeCodeSnapshot,
				TypeLabelSnapshot: o.TypeLabelSnapshot,
				ValueSnapshot:     o.ValueSnapshot,
				PriceSnapshot:     o.PriceSnapshot,
			})
		}
		out = append(out, dto)
	}
	return out
}

func validateOrderReq(r model.OrderCreateRequest) error {
	if len(r.Items) == 0 {
		return model.NewInvalidInput("Заказ должен содержать хотя бы одну позицию")
	}
	if len(r.Items) > 100 {
		return model.NewInvalidInput("Слишком много позиций (максимум 100)")
	}
	for _, it := range r.Items {
		if it.Quantity < 1 || it.Quantity > 99 {
			return model.NewInvalidInput("Количество должно быть от 1 до 99")
		}
		seen := map[uuid.UUID]bool{}
		for _, oid := range it.OptionIDs {
			if seen[oid] {
				return model.NewInvalidInput("Дублирующаяся опция в позиции заказа")
			}
			seen[oid] = true
		}
	}
	if strings.TrimSpace(r.ContactPhone) == "" {
		return model.NewInvalidInput("Укажите телефон для связи")
	}
	if strings.TrimSpace(r.ContactFullName) == "" {
		return model.NewInvalidInput("Укажите имя для связи")
	}
	return nil
}

func isValidStatus(s model.OrderStatus) bool {
	switch s {
	case model.OrderStatusCreated, model.OrderStatusConfirmed, model.OrderStatusManufacturing,
		model.OrderStatusDelivering, model.OrderStatusCompleted, model.OrderStatusCancelled:
		return true
	}
	return false
}
