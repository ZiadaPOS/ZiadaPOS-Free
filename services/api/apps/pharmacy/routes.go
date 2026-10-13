package pharmacy

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
	"github.com/troubleman96/ziadapos/internal/server"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&MedicineProfile{}, &PrescriptionRecord{}, &ExpiryAlertRun{}, &inventory.MedicineStockMovement{}); err != nil {
		return err
	}
	var batches []inventory.MedicineBatch
	if err := db.Where("quantity > 0").Find(&batches).Error; err != nil {
		return err
	}
	for _, batch := range batches {
		var count int64
		if err := db.Model(&inventory.MedicineStockMovement{}).Where("batch_id = ?", batch.ID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			movement := inventory.MedicineStockMovement{StoreID: batch.StoreID, ProductID: batch.ProductID, BatchID: batch.ID, Quantity: batch.Quantity, MovementType: "opening_balance", Reason: "Migrated batch balance"}
			if err := db.Create(&movement).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(deps.DB)
	if deps.Scheduler != nil {
		if err := svc.ScheduleExpiryAlerts(deps.DB, deps.Scheduler); err != nil {
			log.Printf("pharmacy expiry alert scheduler: %v", err)
		}
	}
	batchService := inventory.NewService(inventory.NewRepo(deps.DB), deps.Cache, deps.Bus)
	router := base.Group("/pharmacy", middleware.Auth(deps.Tokens))
	router.GET("/batches", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		productID := uuid.Nil
		if raw := c.Query("product_id"); raw != "" {
			parsed, err := uuid.Parse(raw)
			if err != nil {
				response.BadRequest(c, "Invalid product id", nil)
				return
			}
			productID = parsed
		}
		batches, err := batchService.ListBatches(c.Request.Context(), storeID, productID)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, gin.H{"items": batches})
	})
	router.GET("/batches/:id/movements", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		batchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid batch id", nil)
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
		items, total, err := svc.ListBatchMovements(c.Request.Context(), storeID, batchID, page, pageSize)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 50
		} else if pageSize > 200 {
			pageSize = 200
		}
		response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
	})
	router.POST("/batches/:id/dispose", middleware.RequireOwner(), func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		batchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid batch id", nil)
			return
		}
		userID, err := uuid.Parse(middleware.UserID(c))
		if err != nil {
			response.Unauthorized(c, "Invalid user")
			return
		}
		var input DisposeBatchInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, "Invalid disposal details", nil)
			return
		}
		if err := svc.DisposeBatch(c.Request.Context(), storeID, batchID, userID, input); err != nil {
			pharmacyError(c, err)
			return
		}
		response.Created(c, gin.H{"batch_id": batchID}, "Batch quantity disposed")
	})
	router.POST("/batches/:id/release", middleware.RequireOwner(), func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		batchID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid batch id", nil)
			return
		}
		userID, err := uuid.Parse(middleware.UserID(c))
		if err != nil {
			response.Unauthorized(c, "Invalid user")
			return
		}
		var input ReleaseReturnedBatchInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, "An inspection note is required", nil)
			return
		}
		if err := svc.ReleaseReturnedBatch(c.Request.Context(), storeID, batchID, userID, input); err != nil {
			pharmacyError(c, err)
			return
		}
		batchService.InvalidateProductCache(&storeID, nil)
		response.OK(c, gin.H{"batch_id": batchID, "stock_status": "available"})
	})
	router.POST("/products/:id/batches", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		productID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid product id", nil)
			return
		}
		userID, err := uuid.Parse(middleware.UserID(c))
		if err != nil {
			response.Unauthorized(c, "Invalid user")
			return
		}
		var input inventory.ReceiveBatchInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, "Invalid batch details", nil)
			return
		}
		batch, err := batchService.ReceiveBatch(c.Request.Context(), storeID, productID, userID, input)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.Created(c, batch, "Batch received")
	})
	router.GET("/expiry", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		window, _ := strconv.Atoi(c.DefaultQuery("days", "60"))
		items, err := svc.ExpiryOverview(c.Request.Context(), storeID, window)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, gin.H{"items": items})
	})
	router.GET("/prescriptions", requirePrescriptionRegisterAccess(deps.DB), func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 50
		} else if pageSize > 200 {
			pageSize = 200
		}
		from, err := parsePrescriptionDate(c.Query("from"), false)
		if err != nil {
			response.BadRequest(c, "Invalid from date; expected YYYY-MM-DD", nil)
			return
		}
		to, err := parsePrescriptionDate(c.Query("to"), true)
		if err != nil {
			response.BadRequest(c, "Invalid to date; expected YYYY-MM-DD", nil)
			return
		}
		items, total, err := svc.ListPrescriptionRecords(c.Request.Context(), storeID, c.Query("search"), from, to, page, pageSize)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
	})
	router.GET("/products", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		profiles, err := svc.ListMedicineProfiles(c.Request.Context(), storeID)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, gin.H{"items": profiles})
	})
	router.GET("/products/:id", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		productID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid product id", nil)
			return
		}
		profile, err := svc.GetMedicineProfile(c.Request.Context(), storeID, productID)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, profile)
	})
	router.PUT("/products/:id", func(c *gin.Context) {
		storeID, ok := pharmacyStore(c)
		if !ok {
			return
		}
		productID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.BadRequest(c, "Invalid product id", nil)
			return
		}
		var input MedicineProfileInput
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, "Invalid medicine details", nil)
			return
		}
		profile, err := svc.UpdateMedicineProfile(c.Request.Context(), storeID, productID, input)
		if err != nil {
			pharmacyError(c, err)
			return
		}
		response.OK(c, profile)
	})
}

func requirePrescriptionRegisterAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(middleware.UserID(c))
		if err != nil {
			response.Unauthorized(c, "Invalid user")
			c.Abort()
			return
		}
		var user accounts.User
		if err := db.WithContext(c.Request.Context()).Select("id", "role", "occupation").First(&user, "id = ?", userID).Error; err != nil {
			response.Unauthorized(c, "User not found")
			c.Abort()
			return
		}
		occupation := strings.ToLower(strings.TrimSpace(user.Occupation))
		clinicalRoles := map[string]bool{
			"pharmacist": true, "pharmaceutical_technician": true,
			"pharmaceutical_assistant": true, "medicine_dispenser": true, "dispenser": true,
		}
		if user.Role != accounts.RoleOwner && user.Role != accounts.RoleAdmin && !clinicalRoles[occupation] {
			response.Forbidden(c, "Prescription register access is limited to pharmacy management and pharmaceutical personnel")
			c.Abort()
			return
		}
		c.Next()
	}
}

func parsePrescriptionDate(raw string, endOfDay bool) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	parsed, err := time.ParseInLocation("2006-01-02", raw, location)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.AddDate(0, 0, 1)
	}
	return &parsed, nil
}

func pharmacyStore(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return uuid.Nil, false
	}
	return id, true
}

func pharmacyError(c *gin.Context, err error) {
	typed := apierr.Translate(err)
	response.Error(c, typed.Status, typed.Message, typed.Fields)
}
