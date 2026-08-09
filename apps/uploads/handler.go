// Package uploads stores user-supplied files (product images, logos,
// receipts, attachments) in MinIO and returns stable public URLs that the
// clients save on their records.
package uploads

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
	"github.com/troubleman96/ziadapos/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

// allowedContentTypes maps accepted MIME types to a canonical extension.
var allowedContentTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"application/pdf": ".pdf",
}

// allowedKinds keeps folder naming predictable across clients.
var allowedKinds = map[string]bool{
	"image": true, "logo": true, "receipt": true, "attachment": true,
}

// Upload accepts a multipart file and stores it under the caller's
// organisation prefix. Returns the stable public URL.
func Upload(c *gin.Context, store *storage.Storage) {
	if store == nil {
		response.Error(c, http.StatusServiceUnavailable, "Object storage is not configured", nil)
		return
	}
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Missing 'file' in multipart form", nil)
		return
	}
	defer file.Close()
	if header.Size > maxUploadSize {
		response.BadRequest(c, "File too large", nil)
		return
	}

	kind := c.DefaultPostForm("kind", "image")
	if !allowedKinds[kind] {
		kind = "image"
	}

	// Detect the real content type from the first 512 bytes (never trust the
	// client's declared value alone).
	buf := make([]byte, 512)
	n, _ := io.ReadFull(file, buf)
	buf = buf[:n]
	detected := http.DetectContentType(buf)

	ext, ok := allowedContentTypes[detected]
	if !ok {
		// Fall back to the filename extension for types DetectContentType
		// cannot guess (e.g. application/octet-stream).
		fallback := strings.ToLower(filepath.Ext(header.Filename))
		if fallback == ".jpg" || fallback == ".jpeg" || fallback == ".png" || fallback == ".webp" || fallback == ".gif" || fallback == ".pdf" {
			detected = mime.TypeByExtension(fallback)
			if detected == "" {
				detected = "application/octet-stream"
			}
			ext = fallback
			if ext == ".jpeg" {
				ext = ".jpg"
			}
		} else {
			response.BadRequest(c, "Unsupported file type", map[string]interface{}{
				"file": "allowed: jpg, png, webp, gif, pdf",
			})
			return
		}
	}

	content := make([]byte, 0, header.Size)
	content = append(content, buf...)
	rest, err := io.ReadAll(file)
	if err != nil {
		response.Internal(c, "Failed to read file")
		return
	}
	content = append(content, rest...)

	folder := c.DefaultPostForm("folder", "uploads")
	key := fmt.Sprintf("%s/%s/%s%s", orgID, folder, uuid.New().String(), ext)

	url, err := store.Put(c.Request.Context(), key, content, detected)
	if err != nil {
		response.Internal(c, "Failed to store file")
		return
	}

	response.Created(c, map[string]interface{}{
		"url":          url,
		"key":          key,
		"name":         header.Filename,
		"size":         header.Size,
		"content_type": detected,
		"kind":         kind,
	}, "File uploaded")
}

// Delete removes an object. The key must belong to the caller's organisation
// so users cannot delete each other's files.
func Delete(c *gin.Context, store *storage.Storage) {
	if store == nil {
		response.Error(c, http.StatusServiceUnavailable, "Object storage is not configured", nil)
		return
	}
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	var req struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	prefix := orgID.String() + "/"
	if !strings.HasPrefix(req.Key, prefix) {
		response.Forbidden(c, "Cannot delete this file")
		return
	}
	if err := store.Delete(c.Request.Context(), req.Key); err != nil {
		response.Internal(c, "Failed to delete file")
		return
	}
	response.OK(c, nil)
}
