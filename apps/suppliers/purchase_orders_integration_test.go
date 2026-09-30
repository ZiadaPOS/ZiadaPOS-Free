package suppliers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/apps/pharmacy"
	"github.com/troubleman96/ziadapos/apps/transactions"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPharmacyPurchasingDatabaseWorkflow(t *testing.T) {
	dsn := os.Getenv("ZIADA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZIADA_TEST_DATABASE_URL to run the PostgreSQL workflow integration test")
	}
	db, cleanup := isolatedPostgresSchema(t, dsn)
	defer cleanup()
	for _, migrate := range []func(*gorm.DB) error{
		accounts.Migrate,
		inventory.Migrate,
		pharmacy.Migrate,
		transactions.Migrate,
		Migrate,
	} {
		if err := migrate(db); err != nil {
			t.Fatalf("migration failed: %v", err)
		}
	}

	ctx := context.Background()
	org := accounts.Organisation{Name: "Integration Pharmacy", Slug: "integration-pharmacy", BusinessType: "pharmacy", OutletType: "pharmacy", IsActive: true}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	store := accounts.Store{OrganisationID: org.ID, Name: "Integration Pharmacy Main", IsMainStore: true, Status: "open", IsActive: true}
	if err := db.Create(&store).Error; err != nil {
		t.Fatal(err)
	}
	supplierA := Supplier{OrganisationID: org.ID, StoreID: store.ID, Name: "Supplier A", LeadTimeDays: 5, Status: StatusActive}
	supplierB := Supplier{OrganisationID: org.ID, StoreID: store.ID, Name: "Supplier B", LeadTimeDays: 4, Status: StatusActive}
	if err := db.Create(&supplierA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&supplierB).Error; err != nil {
		t.Fatal(err)
	}
	productA := inventory.Product{StoreID: store.ID, Name: "Medicine A", SellingPrice: 1000, CostPrice: 80, Stock: 18, ReorderPoint: 20, SupplierID: &supplierA.ID, IsActive: true}
	productB := inventory.Product{StoreID: store.ID, Name: "Medicine B", SellingPrice: 2000, CostPrice: 40, ReorderPoint: 0, SupplierID: &supplierB.ID, IsActive: true}
	if err := db.Create(&productA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&productB).Error; err != nil {
		t.Fatal(err)
	}
	batch := inventory.MedicineBatch{StoreID: store.ID, ProductID: productA.ID, BatchNumber: "A-OPEN", Quantity: 18, ExpiryDate: time.Now().AddDate(0, 0, 120), CostPrice: 80}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&inventory.MedicineStockMovement{StoreID: store.ID, ProductID: productA.ID, BatchID: batch.ID, Quantity: 18, MovementType: "opening_balance"}).Error; err != nil {
		t.Fatal(err)
	}
	auditItems, auditTotal, err := pharmacy.NewService(db).ListBatchMovements(ctx, store.ID, batch.ID, 1, 50)
	if err != nil || auditTotal != 1 || len(auditItems) != 1 || auditItems[0].Quantity != 18 || auditItems[0].MovementType != "opening_balance" {
		t.Fatalf("batch movement audit failed: total=%d items=%+v err=%v", auditTotal, auditItems, err)
	}
	checkout := transactions.NewService(transactions.NewRepo(db), nil, nil, nil, db, nil)
	_, duplicateErr := checkout.CompleteSale(ctx, org.ID, store.ID, uuid.New(), &transactions.SaleInput{
		Lines: []transactions.SaleLineInput{{ProductID: productA.ID, Qty: 1}, {ProductID: productA.ID, Qty: 1}}, PaymentMethod: transactions.MethodCash,
	})
	if duplicateErr == nil || !strings.Contains(duplicateErr.Error(), "Each product may appear only once") {
		t.Fatalf("sale with duplicate medicine lines should be rejected, got %v", duplicateErr)
	}
	_, saleErr := checkout.CompleteSale(ctx, org.ID, store.ID, uuid.New(), &transactions.SaleInput{
		Lines: []transactions.SaleLineInput{{ProductID: productA.ID, Qty: 1}}, PaymentMethod: transactions.MethodCash,
	})
	if saleErr == nil || !strings.Contains(saleErr.Error(), "Medicine details must be configured") {
		t.Fatalf("sale without a medicine profile should be rejected, got %v", saleErr)
	}
	transaction := transactions.Transaction{OrganisationID: org.ID, StoreID: store.ID, TxnNumber: "INT-0001", PaymentMethod: transactions.MethodCash, Status: transactions.StatusPaid, CashierID: uuid.New(), Subtotal: 300000, Total: 300000}
	transaction.CreatedAt = time.Now().AddDate(0, 0, -15)
	if err := db.Create(&transaction).Error; err != nil {
		t.Fatal(err)
	}
	productAID := productA.ID
	saleLine := transactions.TransactionLine{TransactionID: transaction.ID, ProductID: &productAID, ProductName: productA.Name, UnitPrice: 1000, UnitCost: 80, Qty: 300}
	if err := db.Create(&saleLine).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pharmacy.PrescriptionRecord{StoreID: store.ID, TransactionLineID: saleLine.ID, PrescriberName: "Dr Integration", PrescriptionReference: "RX-INT-007", PatientName: "Patient Integration", RecordedBy: uuid.New()}).Error; err != nil {
		t.Fatal(err)
	}
	rxItems, rxTotal, err := pharmacy.NewService(db).ListPrescriptionRecords(ctx, store.ID, "RX-INT-007", nil, nil, 1, 50)
	if err != nil || rxTotal != 1 || len(rxItems) != 1 || rxItems[0].ProductName != productA.Name {
		t.Fatalf("prescription audit query failed: total=%d items=%+v err=%v", rxTotal, rxItems, err)
	}

	svc := NewService(NewRepo(db), nil, nil)
	suggestions, err := svc.ReorderSuggestions(ctx, org.ID, store.ID)
	if err != nil {
		t.Fatalf("reorder suggestions: %v", err)
	}
	var productASuggestion *ReorderSuggestion
	for i := range suggestions {
		if suggestions[i].ProductID == productA.ID {
			productASuggestion = &suggestions[i]
		}
	}
	if productASuggestion == nil || productASuggestion.SuggestedQuantity != 52 {
		t.Fatalf("unexpected suggestions: %+v; want Medicine A with quantity 52", suggestions)
	}

	invalidGroups := CreatePurchaseOrdersInput{Groups: []PurchaseOrderSupplierGroup{
		{SupplierID: supplierA.ID, Lines: []PurchaseOrderLineInput{{ProductID: productA.ID, Quantity: 10, UnitCost: 80}}},
		{SupplierID: uuid.New(), Lines: []PurchaseOrderLineInput{{ProductID: productB.ID, Quantity: 5, UnitCost: 40}}},
	}}
	if _, err := svc.CreatePurchaseOrders(ctx, org.ID, store.ID, uuid.New(), invalidGroups); err == nil {
		t.Fatal("expected invalid supplier group to roll back the bulk order request")
	}
	orders, err := svc.ListPurchaseOrders(ctx, store.ID)
	if err != nil || len(orders) != 0 {
		t.Fatalf("invalid bulk request left persisted purchase orders: count=%d err=%v", len(orders), err)
	}

	created, err := svc.CreatePurchaseOrders(ctx, org.ID, store.ID, uuid.New(), CreatePurchaseOrdersInput{Groups: []PurchaseOrderSupplierGroup{
		{SupplierID: supplierA.ID, Lines: []PurchaseOrderLineInput{{ProductID: productA.ID, Quantity: 10, UnitCost: 80}}},
		{SupplierID: supplierB.ID, Lines: []PurchaseOrderLineInput{{ProductID: productB.ID, Quantity: 5, UnitCost: 40}}},
	}})
	if err != nil {
		t.Fatalf("create grouped purchase orders: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created %d orders; want 2 supplier orders", len(created))
	}
	var orderB *PurchaseOrder
	for i := range created {
		if created[i].SupplierID == supplierB.ID {
			orderB = &created[i]
		}
	}
	if orderB == nil || len(orderB.Lines) != 1 {
		t.Fatalf("missing Supplier B purchase order: %+v", created)
	}
	received, err := svc.ReceivePurchaseOrder(ctx, org.ID, store.ID, uuid.New(), orderB.ID, ReceivePurchaseOrderInput{InvoiceNumber: "INV-INT-01", Lines: []PurchaseReceiptLineInput{{LineID: orderB.Lines[0].ID, Quantity: 3, BatchNumber: "B-001", ExpiryDate: time.Now().AddDate(0, 0, 150).Format("2006-01-02")}}})
	if err != nil {
		t.Fatalf("partial purchase receipt: %v", err)
	}
	if received.Status != PurchaseOrderPartial || received.Lines[0].ReceivedQuantity != 3 {
		t.Fatalf("partial receipt state is incorrect: status=%s lines=%+v", received.Status, received.Lines)
	}
	var persistedProduct inventory.Product
	if err := db.First(&persistedProduct, "id = ?", productB.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persistedProduct.Stock != 3 {
		t.Fatalf("product stock=%d after receiving 3; want 3", persistedProduct.Stock)
	}
	var persistedBatch inventory.MedicineBatch
	if err := db.Where("store_id = ? AND product_id = ? AND batch_number = ?", store.ID, productB.ID, "B-001").First(&persistedBatch).Error; err != nil {
		t.Fatal(err)
	}
	if persistedBatch.Quantity != 3 {
		t.Fatalf("batch quantity=%d; want 3", persistedBatch.Quantity)
	}
	var movementTotal int64
	if err := db.Model(&inventory.MedicineStockMovement{}).Where("batch_id = ?", persistedBatch.ID).Select("COALESCE(SUM(quantity), 0)").Scan(&movementTotal).Error; err != nil {
		t.Fatal(err)
	}
	if movementTotal != 3 {
		t.Fatalf("movement ledger total=%d; want 3", movementTotal)
	}
}

func isolatedPostgresSchema(t *testing.T, dsn string) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open database pool: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := "ziada_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := db.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	if err := db.Exec(fmt.Sprintf("SET search_path TO \"%s\"", schema)).Error; err != nil {
		t.Fatalf("set isolated test schema: %v", err)
	}
	return db, func() {
		if err := db.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		_ = sqlDB.Close()
	}
}
