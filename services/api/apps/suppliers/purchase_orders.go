package suppliers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/response"
)

type PurchaseOrderLineInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,gt=0"`
	UnitCost  int64     `json:"unit_cost" binding:"gte=0"`
}

type CreatePurchaseOrderInput struct {
	SupplierID uuid.UUID                `json:"supplier_id" binding:"required"`
	Notes      string                   `json:"notes"`
	Lines      []PurchaseOrderLineInput `json:"lines" binding:"required,min=1,dive"`
}

type PurchaseOrderSupplierGroup struct {
	SupplierID uuid.UUID                `json:"supplier_id" binding:"required"`
	Lines      []PurchaseOrderLineInput `json:"lines" binding:"required,min=1,dive"`
}

type CreatePurchaseOrdersInput struct {
	Notes  string                       `json:"notes"`
	Groups []PurchaseOrderSupplierGroup `json:"groups" binding:"required,min=1,dive"`
}

type PurchaseReceiptLineInput struct {
	LineID      uuid.UUID `json:"line_id" binding:"required"`
	Quantity    int       `json:"quantity" binding:"required,gt=0"`
	BatchNumber string    `json:"batch_number" binding:"required"`
	ExpiryDate  string    `json:"expiry_date" binding:"required"`
}

type ReceivePurchaseOrderInput struct {
	InvoiceNumber string                     `json:"invoice_number"`
	Lines         []PurchaseReceiptLineInput `json:"lines" binding:"required,min=1,dive"`
}

func (s *Service) CreatePurchaseOrder(ctx context.Context, orgID, storeID, userID uuid.UUID, in CreatePurchaseOrderInput) (*PurchaseOrder, error) {
	if _, err := s.repo.Get(storeID, in.SupplierID); err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	var businessType string
	if err := s.repo.DB().WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType != "pharmacy" {
		return nil, apierr.Conflict("Purchase orders are available for pharmacy accounts")
	}
	if len(in.Lines) == 0 {
		return nil, apierr.BadRequest("Add at least one product to the purchase order")
	}
	po := &PurchaseOrder{StoreID: storeID, OrganisationID: orgID, SupplierID: in.SupplierID, OrderNumber: fmt.Sprintf("PO-%s-%s", time.Now().Format("20060102"), strings.ToUpper(uuid.NewString()[:6])), Status: PurchaseOrderOrdered, Notes: strings.TrimSpace(in.Notes), CreatedBy: userID}
	db := s.repo.DB().WithContext(ctx)
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(po).Error; err != nil {
			return err
		}
		seen := map[uuid.UUID]bool{}
		for _, line := range in.Lines {
			if line.Quantity <= 0 || line.UnitCost < 0 || seen[line.ProductID] {
				return apierr.BadRequest("Purchase order lines need unique products, positive quantities, and nonnegative costs")
			}
			seen[line.ProductID] = true
			var product inventory.Product
			if err := tx.Where("id = ? AND store_id = ? AND is_active = true AND is_service = false", line.ProductID, storeID).First(&product).Error; err != nil {
				return apierr.BadRequest("Purchase order product is unavailable in this store").WithField("product_id", line.ProductID.String())
			}
			if err := tx.Create(&PurchaseOrderLine{PurchaseOrderID: po.ID, ProductID: line.ProductID, OrderedQuantity: line.Quantity, UnitCost: line.UnitCost}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := db.Preload("Lines").First(po, "id = ?", po.ID).Error; err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	return po, nil
}

func (s *Service) CreatePurchaseOrders(ctx context.Context, orgID, storeID, userID uuid.UUID, in CreatePurchaseOrdersInput) ([]PurchaseOrder, error) {
	if len(in.Groups) == 0 {
		return nil, apierr.BadRequest("Select at least one reorder suggestion")
	}
	var businessType string
	if err := s.repo.DB().WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType != "pharmacy" {
		return nil, apierr.Conflict("Purchase orders are available for pharmacy accounts")
	}
	db := s.repo.DB().WithContext(ctx)
	created := make([]PurchaseOrder, 0, len(in.Groups))
	globalProducts := make(map[uuid.UUID]bool)
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, group := range in.Groups {
			if group.SupplierID == uuid.Nil || len(group.Lines) == 0 {
				return apierr.BadRequest("Each purchase order group needs a supplier and at least one product")
			}
			var supplier Supplier
			if err := tx.Where("id = ? AND store_id = ? AND status = ?", group.SupplierID, storeID, StatusActive).First(&supplier).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return apierr.BadRequest("Selected supplier is unavailable")
				}
				return err
			}
			order := PurchaseOrder{StoreID: storeID, OrganisationID: orgID, SupplierID: supplier.ID, OrderNumber: fmt.Sprintf("PO-%s-%s", time.Now().Format("20060102"), strings.ToUpper(uuid.NewString()[:6])), Status: PurchaseOrderOrdered, Notes: strings.TrimSpace(in.Notes), CreatedBy: userID}
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			groupProducts := make(map[uuid.UUID]bool)
			for _, line := range group.Lines {
				if line.ProductID == uuid.Nil || line.Quantity <= 0 || line.UnitCost < 0 || groupProducts[line.ProductID] || globalProducts[line.ProductID] {
					return apierr.BadRequest("Bulk order lines need unique products, positive quantities, and nonnegative costs")
				}
				groupProducts[line.ProductID] = true
				globalProducts[line.ProductID] = true
				var product inventory.Product
				if err := tx.Where("id = ? AND store_id = ? AND is_active = true AND is_service = false", line.ProductID, storeID).First(&product).Error; err != nil {
					return apierr.BadRequest("Purchase order product is unavailable in this store").WithField("product_id", line.ProductID.String())
				}
				if err := tx.Create(&PurchaseOrderLine{PurchaseOrderID: order.ID, ProductID: line.ProductID, OrderedQuantity: line.Quantity, UnitCost: line.UnitCost}).Error; err != nil {
					return err
				}
			}
			created = append(created, order)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range created {
		if err := db.Preload("Lines").First(&created[i], "id = ?", created[i].ID).Error; err != nil {
			return nil, err
		}
	}
	s.invalidateCache(storeID)
	return created, nil
}

func (s *Service) ListPurchaseOrders(ctx context.Context, storeID uuid.UUID) ([]PurchaseOrder, error) {
	var orders []PurchaseOrder
	err := s.repo.DB().WithContext(ctx).Preload("Lines").Where("store_id = ?", storeID).Order("created_at desc").Limit(200).Find(&orders).Error
	return orders, err
}

func (s *Service) ReceivePurchaseOrder(ctx context.Context, orgID, storeID, userID, orderID uuid.UUID, in ReceivePurchaseOrderInput) (*PurchaseOrder, error) {
	db := s.repo.DB().WithContext(ctx)
	var productIDs []uuid.UUID
	err := db.Transaction(func(tx *gorm.DB) error {
		var order PurchaseOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ? AND organisation_id = ?", orderID, storeID, orgID).First(&order).Error; err != nil {
			return apierr.NotFound("Purchase order not found")
		}
		if order.Status == PurchaseOrderReceived {
			return apierr.Conflict("Purchase order is already fully received")
		}
		var lines []PurchaseOrderLine
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("purchase_order_id = ?", order.ID).Find(&lines).Error; err != nil {
			return err
		}
		byID := make(map[uuid.UUID]*PurchaseOrderLine, len(lines))
		for i := range lines {
			byID[lines[i].ID] = &lines[i]
		}
		var supplier Supplier
		if err := tx.Where("id = ? AND store_id = ?", order.SupplierID, storeID).First(&supplier).Error; err != nil {
			return err
		}
		delivery := SupplierDelivery{SupplierID: order.SupplierID, PurchaseOrderID: &order.ID, StoreID: storeID, OrganisationID: orgID, InvoiceNumber: strings.TrimSpace(in.InvoiceNumber), DeliveryDate: time.Now(), Status: DeliveryDelivered, RecordedBy: userID, Notes: "Receipt for " + order.OrderNumber}
		var total int64
		deliveryLines := make([]SupplierDeliveryLine, 0, len(in.Lines))
		for _, receipt := range in.Lines {
			line, ok := byID[receipt.LineID]
			if !ok {
				return apierr.BadRequest("Receipt line does not belong to this purchase order")
			}
			if receipt.Quantity <= 0 || receipt.Quantity > line.OrderedQuantity-line.ReceivedQuantity {
				return apierr.Conflict("Received quantity exceeds the remaining ordered quantity").WithField("line_id", receipt.LineID.String())
			}
			batchNumber := strings.TrimSpace(receipt.BatchNumber)
			expiry, parseErr := inventory.ParseMedicineExpiryDate(receipt.ExpiryDate, time.Now())
			if batchNumber == "" || parseErr != nil {
				if parseErr != nil {
					return parseErr
				}
				return apierr.BadRequest("Each received pharmacy batch needs a batch number and expiry date")
			}
			var product inventory.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", line.ProductID, storeID).First(&product).Error; err != nil {
				return err
			}
			batch := inventory.MedicineBatch{StoreID: storeID, ProductID: product.ID, BatchNumber: batchNumber, Quantity: 0, ExpiryDate: expiry, CostPrice: line.UnitCost, SupplierName: supplier.Name}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}
			if err := tx.Create(&inventory.MedicineStockMovement{StoreID: storeID, ProductID: product.ID, BatchID: batch.ID, Quantity: receipt.Quantity, MovementType: "purchase", ReferenceID: order.ID.String(), Reason: "Received " + order.OrderNumber, PerformedBy: userID}).Error; err != nil {
				return err
			}
			if err := inventory.ReconcilePharmacyStock(tx, storeID, product.ID, 0); err != nil {
				return err
			}
			if err := tx.Create(&inventory.StockAdjustment{ProductID: product.ID, MedicineBatchID: &batch.ID, QtyChange: receipt.Quantity, Reason: inventory.ReasonRestock, Note: "Received " + order.OrderNumber, CreatedBy: userID}).Error; err != nil {
				return err
			}
			if err := tx.Model(line).Update("received_quantity", gorm.Expr("received_quantity + ?", receipt.Quantity)).Error; err != nil {
				return err
			}
			delivery.ItemsCount = len(in.Lines)
			total += line.UnitCost * int64(receipt.Quantity)
			productIDs = append(productIDs, product.ID)
			line.ReceivedQuantity += receipt.Quantity
			deliveryLines = append(deliveryLines, SupplierDeliveryLine{ProductID: product.ID, MedicineBatchID: &batch.ID, Quantity: receipt.Quantity, UnitCost: line.UnitCost, BatchNumber: batchNumber, ExpiryDate: &expiry})
		}
		delivery.TotalValue = total
		if err := tx.Create(&delivery).Error; err != nil {
			return err
		}
		for i := range deliveryLines {
			deliveryLines[i].DeliveryID = delivery.ID
			if err := tx.Create(&deliveryLines[i]).Error; err != nil {
				return err
			}
		}
		allReceived := true
		for _, line := range lines {
			if line.ReceivedQuantity < line.OrderedQuantity {
				allReceived = false
			}
		}
		order.Status = PurchaseOrderPartial
		if allReceived {
			order.Status = PurchaseOrderReceived
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(productIDs) > 0 {
		inventory.NewService(inventory.NewRepo(db), s.cache, s.bus).InvalidateProductCache(&storeID, &orgID, productIDs...)
	}
	s.invalidateCache(storeID)
	var order PurchaseOrder
	if err := db.Preload("Lines").First(&order, "id = ?", orderID).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (h *Handler) PurchaseOrders(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	items, err := h.svc.ListPurchaseOrders(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *Handler) ReorderSuggestions(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	items, err := h.svc.ReorderSuggestions(c.Request.Context(), orgID, storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"items": items})
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var input CreatePurchaseOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid purchase order", nil)
		return
	}
	order, err := h.svc.CreatePurchaseOrder(c.Request.Context(), orgID, storeID, h.userID(c), input)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, order, "Purchase order created")
}

func (h *Handler) CreatePurchaseOrders(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var input CreatePurchaseOrdersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid grouped purchase orders", nil)
		return
	}
	orders, err := h.svc.CreatePurchaseOrders(c.Request.Context(), orgID, storeID, h.userID(c), input)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, gin.H{"items": orders}, "Purchase orders created")
}

func (h *Handler) ReceivePurchaseOrder(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid purchase order id", nil)
		return
	}
	var input ReceivePurchaseOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid receipt", nil)
		return
	}
	order, err := h.svc.ReceivePurchaseOrder(c.Request.Context(), orgID, storeID, h.userID(c), orderID, input)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, order)
}
