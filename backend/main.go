package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		log.Fatalf("Migration check failed: %v", err)
	}

	userRepo := NewUserRepository(db)
	memoryRepo := NewMemoryRepository(db)
	shareRepo := NewShareRepository(db)

	jwtSvc := NewJWTService(os.Getenv("JWT_SECRET"))
	mediaSvc := NewMediaService(
		os.Getenv("SUPABASE_URL"),
		os.Getenv("SUPABASE_SERVICE_KEY"),
		os.Getenv("SUPABASE_BUCKET"),
	)
	aiSvc := NewAIService(
		os.Getenv("AI_SERVICE_URL"),
		os.Getenv("GEMINI_API_KEY"),
	)
	socialSvc := NewSocialService()

	authH := NewAuthHandler(userRepo, jwtSvc)
	memoryH := NewMemoryHandler(memoryRepo, mediaSvc, aiSvc, socialSvc)
	shareH := NewShareHandler(shareRepo, memoryRepo)
	mediaH := NewMediaHandler(mediaSvc, memoryRepo)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Pin-Token"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "app": "Pikyon API"})
	})

	// Public routes
	pub := r.Group("/api/v1")
	pub.POST("/auth/register", authH.Register)
	pub.POST("/auth/login", authH.Login)
	pub.GET("/memories/shared/:shareToken", shareH.GetSharedMemory)

	// Protected routes
	prot := r.Group("/api/v1")
	prot.Use(JWTMiddleware(jwtSvc))

	// Auth
	prot.GET("/auth/me", authH.Me)
	prot.POST("/auth/pin/set", authH.SetPIN)
	prot.POST("/auth/pin/verify", authH.VerifyPIN)

	// Memories
	prot.GET("/memories", memoryH.ListMemories)
	prot.POST("/memories", memoryH.CreateMemory)
	prot.GET("/memories/:id", memoryH.GetMemory)
	prot.PATCH("/memories/:id", memoryH.UpdateMemory)
	prot.DELETE("/memories/:id", memoryH.DeleteMemory)
	prot.POST("/memories/:id/ai/analyze", memoryH.AnalyzeMemory)

	// Media
	prot.POST("/media/upload-url", mediaH.GetUploadURL)
	prot.POST("/media/confirm", mediaH.ConfirmUpload)
	prot.GET("/media/:memoryId/stream", mediaH.GetStreamURL)

	// Sharing
	prot.POST("/memories/:id/share", shareH.CreateShareLink)
	prot.DELETE("/memories/:id/share", shareH.RevokeShareLink)
	prot.GET("/shared", shareH.ListSharedWithMe)

	// Private lock
	lock := prot.Group("/memories/:id/private")
	lock.Use(PrivateLockMiddleware())
	lock.GET("/unlock", memoryH.UnlockPrivateMemory)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Pikyon API running on :%s", port)
	log.Fatal(r.Run(":" + port))
}
