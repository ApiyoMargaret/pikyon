package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// AUTH HANDLER
// ============================================================

type AuthHandler struct {
	userRepo *UserRepository
	jwtSvc   *JWTService
}

func NewAuthHandler(userRepo *UserRepository, jwtSvc *JWTService) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, jwtSvc: jwtSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required"`
		Lang     string `json:"lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Lang == "" {
		req.Lang = "en"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user, err := h.userRepo.Create(c.Request.Context(), req.Email, req.Name, string(hash), req.Lang)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}
	token, _ := h.jwtSvc.GenerateToken(user.ID, user.Email)
	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"preferred_lang": user.PreferredLang,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	token, _ := h.jwtSvc.GenerateToken(user.ID, user.Email)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"name":           user.Name,
			"preferred_lang": user.PreferredLang,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": gin.H{
		"id":             user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"preferred_lang": user.PreferredLang,
		"has_pin":        user.PINHash.Valid,
	}})
}

func (h *AuthHandler) SetPIN(c *gin.Context) {
	var req struct {
		PIN string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString(UserIDKey)
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err := h.userRepo.SetPIN(c.Request.Context(), userID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set PIN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "PIN set successfully"})
}

func (h *AuthHandler) VerifyPIN(c *gin.Context) {
	var req struct {
		PIN      string `json:"pin" binding:"required"`
		MemoryID string `json:"memory_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString(UserIDKey)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil || !user.PINHash.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No PIN configured"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PINHash.String), []byte(req.PIN)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Incorrect PIN"})
		return
	}
	pinToken, err := h.jwtSvc.GeneratePINToken(userID, req.MemoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PIN token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pin_token":  pinToken,
		"expires_in": 300,
		"memory_id":  req.MemoryID,
	})
}

// ============================================================
// MEMORY HANDLER
// ============================================================

type MemoryHandler struct {
	memoryRepo    *MemoryRepository
	mediaService  *MediaService
	aiService     *AIService
	socialService *SocialService
}

func NewMemoryHandler(
	memoryRepo *MemoryRepository,
	mediaSvc *MediaService,
	aiSvc *AIService,
	socialSvc *SocialService,
) *MemoryHandler {
	return &MemoryHandler{
		memoryRepo:    memoryRepo,
		mediaService:  mediaSvc,
		aiService:     aiSvc,
		socialService: socialSvc,
	}
}

func (h *MemoryHandler) ListMemories(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memories, err := h.memoryRepo.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch memories"})
		return
	}
	if memories == nil {
		memories = []*Memory{}
	}
	c.JSON(http.StatusOK, gin.H{"memories": memories, "count": len(memories)})
}

func (h *MemoryHandler) GetMemory(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	memory, err := h.memoryRepo.GetByID(c.Request.Context(), userID, memoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch memory"})
		return
	}
	if memory.IsLocked {
		c.JSON(http.StatusOK, gin.H{
			"memory": gin.H{
				"id":         memory.ID,
				"title":      memory.Title,
				"is_locked":  true,
				"created_at": memory.CreatedAt,
			},
			"locked": true,
			"action": "pin_required",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"memory": memory})
}

func (h *MemoryHandler) CreateMemory(c *gin.Context) {
	var req struct {
		Title      string   `json:"title" binding:"required"`
		Story      *string  `json:"story"`
		Location   *string  `json:"location"`
		MemoryDate *string  `json:"memory_date"`
		Visibility string   `json:"visibility"`
		IsLocked   bool     `json:"is_locked"`
		MediaKey   *string  `json:"media_key"`
		MediaType  string   `json:"media_type"`
		Tags       []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.MediaType == "" {
		req.MediaType = "none"
	}

	var memoryDate *time.Time
	if req.MemoryDate != nil && *req.MemoryDate != "" {
		t, err := time.Parse("2006-01-02", *req.MemoryDate)
		if err == nil {
			memoryDate = &t
		}
	}

	userID := c.GetString(UserIDKey)
	memory, err := h.memoryRepo.Create(c.Request.Context(), CreateMemoryInput{
		UserID:     userID,
		Title:      req.Title,
		Story:      req.Story,
		Location:   req.Location,
		MemoryDate: memoryDate,
		Visibility: req.Visibility,
		IsLocked:   req.IsLocked,
		MediaKey:   req.MediaKey,
		MediaType:  req.MediaType,
		Tags:       req.Tags,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create memory: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"memory": memory})
}

func (h *MemoryHandler) UpdateMemory(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.memoryRepo.Update(c.Request.Context(), userID, memoryID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update memory"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Memory updated"})
}

func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	memory, _ := h.memoryRepo.GetByIDWithKey(c.Request.Context(), userID, memoryID)
	if err := h.memoryRepo.Delete(c.Request.Context(), userID, memoryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete memory"})
		return
	}
	if memory != nil && memory.MediaKey.Valid && memory.MediaKey.String != "" {
		go h.mediaService.DeleteMedia(memory.MediaKey.String)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Memory deleted"})
}

func (h *MemoryHandler) AnalyzeMemory(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	memory, err := h.memoryRepo.GetByID(c.Request.Context(), userID, memoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
		return
	}
	story := ""
	if memory.Story.Valid {
		story = memory.Story.String
	}
	result, err := h.aiService.AnalyzeMemory(memory.Title, story)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI analysis failed: " + err.Error()})
		return
	}
	caption := ""
	if cap, ok := result.Captions["instagram"]; ok {
		caption = cap
	}
	go h.memoryRepo.UpdateAI(c.Request.Context(), userID, memoryID,
		result.Vibe, result.VibeReason, caption, result.Tone)
	c.JSON(http.StatusOK, gin.H{"analysis": result})
}

func (h *MemoryHandler) UnlockPrivateMemory(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	memory, err := h.memoryRepo.GetByID(c.Request.Context(), userID, memoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"memory": memory, "unlocked": true})
}

// ============================================================
// SHARE HANDLER
// ============================================================

type ShareHandler struct {
	shareRepo  *ShareRepository
	memoryRepo *MemoryRepository
}

func NewShareHandler(shareRepo *ShareRepository, memoryRepo *MemoryRepository) *ShareHandler {
	return &ShareHandler{shareRepo: shareRepo, memoryRepo: memoryRepo}
}

func (h *ShareHandler) CreateShareLink(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	sa, err := h.shareRepo.Create(c.Request.Context(), memoryID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create share link"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"share_token": sa.ShareToken,
		"share_url":   "/shared/" + sa.ShareToken,
	})
}

func (h *ShareHandler) RevokeShareLink(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("id")
	h.shareRepo.Delete(c.Request.Context(), memoryID, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Share link revoked"})
}

func (h *ShareHandler) GetSharedMemory(c *gin.Context) {
	shareToken := c.Param("shareToken")
	sa, err := h.shareRepo.GetByToken(c.Request.Context(), shareToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}
	memory, err := h.memoryRepo.GetByID(c.Request.Context(), sa.OwnerID, sa.MemoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory not available"})
		return
	}
	go h.shareRepo.IncrementViewCount(c.Request.Context(), shareToken)
	c.JSON(http.StatusOK, gin.H{"memory": memory})
}

func (h *ShareHandler) ListSharedWithMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"shared": []interface{}{}})
}

// ============================================================
// MEDIA HANDLER
// ============================================================

type MediaHandler struct {
	mediaService *MediaService
	memoryRepo   *MemoryRepository
}

func NewMediaHandler(mediaSvc *MediaService, memoryRepo *MemoryRepository) *MediaHandler {
	return &MediaHandler{mediaService: mediaSvc, memoryRepo: memoryRepo}
}

func (h *MediaHandler) GetStreamURL(c *gin.Context) {
	userID := c.GetString(UserIDKey)
	memoryID := c.Param("memoryId")
	memory, err := h.memoryRepo.GetByIDWithKey(c.Request.Context(), userID, memoryID)
	if err != nil || !memory.HasMedia {
		c.JSON(http.StatusNotFound, gin.H{"error": "No media found"})
		return
	}
	result, err := h.mediaService.GetStreamURL(memory.MediaKey.String)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate stream URL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"signed_url": result.SignedURL,
		"expires_at": result.ExpiresAt,
		"media_type": memory.MediaType,
	})
}

func (h *MediaHandler) GetUploadURL(c *gin.Context) {
	var req struct {
		MemoryID string `json:"memory_id" binding:"required"`
		FileExt  string `json:"file_ext" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString(UserIDKey)
	result, err := h.mediaService.GetUploadURL(userID, req.MemoryID, req.FileExt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"upload_url": result.UploadURL,
		"media_key":  result.MediaKey,
	})
}

func (h *MediaHandler) ConfirmUpload(c *gin.Context) {
	var req struct {
		MemoryID  string `json:"memory_id" binding:"required"`
		MediaKey  string `json:"media_key" binding:"required"`
		MediaType string `json:"media_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString(UserIDKey)
	if err := h.memoryRepo.UpdateMediaKey(c.Request.Context(), userID, req.MemoryID, req.MediaKey, req.MediaType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm upload"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Media confirmed successfully"})
}