package handle

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"backend/internal/repo"

	"github.com/gin-gonic/gin"
)

type SitemapHandler struct {
    tagRepo      *repo.TagRepo
    categoryRepo *repo.CategoryRepo
    articleRepo  *repo.ArticleRepo
}

func NewSitemapHandler() *SitemapHandler {
    return &SitemapHandler{
        tagRepo:      repo.NewTagRepo(),
        categoryRepo: repo.NewCategoryRepo(),
        articleRepo:  repo.NewArticleRepo(),
    }
}

// SitemapURL định dạng JSON trả về cho Nuxt
type SitemapURL struct {
    Loc        string  `json:"loc"`
    LastMod    string  `json:"lastmod,omitempty"`
    ChangeFreq string  `json:"changefreq,omitempty"`
    Priority   float64 `json:"priority,omitempty"`
}

// getPublicBase lấy domain public từ env PUBLIC_WEB_DOMAIN (ví dụ https://quantriduanxaydung.vn)
// fallback về https://quantriduanxaydung.vn nếu không set
func getPublicBase() string {
    v := os.Getenv("PUBLIC_WEB_DOMAIN")
    if v == "" {
        return "https://quantriduanxaydung.vn"
    }
    if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
        return strings.TrimRight(v, "/")
    }
    return "https://" + strings.TrimRight(v, "/")
}

// Simple in-memory caches per resource to avoid hitting DB on every request
var (
    tagsCache struct{
        data []SitemapURL
        expires time.Time
    }
    categoriesCache struct{
        data []SitemapURL
        expires time.Time
    }
    articlesCache struct{
        data []SitemapURL
        expires time.Time
    }
    cacheMu sync.RWMutex
)

func cacheTTL() time.Duration {
    if v := os.Getenv("SITEMAP_CACHE_TTL_SECONDS"); v != "" {
        if s, err := time.ParseDuration(v + "s"); err == nil {
            return s
        }
    }
    return time.Hour
}

// GetTagsURLs trả về SitemapURL cho các tags
func (h *SitemapHandler) GetTagsURLs(c *gin.Context) {
    cacheMu.RLock()
    if time.Now().Before(tagsCache.expires) && tagsCache.data != nil {
        data := tagsCache.data
        cacheMu.RUnlock()
        c.Header("Cache-Control", "public, max-age=3600")
        c.JSON(http.StatusOK, data)
        return
    }
    cacheMu.RUnlock()

    rows, err := h.tagRepo.GetAllSlugsWithUpdatedAt(true)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    base := getPublicBase()
    out := make([]SitemapURL, 0, len(rows))
    for _, r := range rows {
        out = append(out, SitemapURL{
            Loc: base + "/tags/" + r.Slug,
            LastMod: r.UpdatedAt.Format("2006-01-02"),
            ChangeFreq: "weekly",
            Priority: 0.6,
        })
    }

    cacheMu.Lock()
    tagsCache.data = out
    tagsCache.expires = time.Now().Add(cacheTTL())
    cacheMu.Unlock()

    c.Header("Cache-Control", "public, max-age=3600")
    c.JSON(http.StatusOK, out)
}

// GetCategoriesURLs trả về SitemapURL cho các categories
func (h *SitemapHandler) GetCategoriesURLs(c *gin.Context) {
    cacheMu.RLock()
    if time.Now().Before(categoriesCache.expires) && categoriesCache.data != nil {
        data := categoriesCache.data
        cacheMu.RUnlock()
        c.Header("Cache-Control", "public, max-age=3600")
        c.JSON(http.StatusOK, data)
        return
    }
    cacheMu.RUnlock()

    rows, err := h.categoryRepo.GetAllSlugsWithUpdatedAt(true)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    base := getPublicBase()
    out := make([]SitemapURL, 0, len(rows))
    for _, r := range rows {
        out = append(out, SitemapURL{
            Loc: base + "/categories/" + r.Slug,
            LastMod: r.UpdatedAt.Format("2006-01-02"),
            ChangeFreq: "weekly",
            Priority: 0.7,
        })
    }

    cacheMu.Lock()
    categoriesCache.data = out
    categoriesCache.expires = time.Now().Add(cacheTTL())
    cacheMu.Unlock()

    c.Header("Cache-Control", "public, max-age=3600")
    c.JSON(http.StatusOK, out)
}

// GetArticlesURLs trả về SitemapURL cho các bài viết đã xuất bản
func (h *SitemapHandler) GetArticlesURLs(c *gin.Context) {
    cacheMu.RLock()
    if time.Now().Before(articlesCache.expires) && articlesCache.data != nil {
        data := articlesCache.data
        cacheMu.RUnlock()
        c.Header("Cache-Control", "public, max-age=3600")
        c.JSON(http.StatusOK, data)
        return
    }
    cacheMu.RUnlock()

    rows, err := h.articleRepo.GetPublishedSlugsWithUpdatedAt(0)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    base := getPublicBase()
    out := make([]SitemapURL, 0, len(rows))
    for _, r := range rows {
        out = append(out, SitemapURL{
            Loc: base + "/bai-viet/" + r.Slug,
            LastMod: r.UpdatedAt.Format("2006-01-02"),
            ChangeFreq: "weekly",
            Priority: 0.8,
        })
    }

    cacheMu.Lock()
    articlesCache.data = out
    articlesCache.expires = time.Now().Add(cacheTTL())
    cacheMu.Unlock()

    c.Header("Cache-Control", "public, max-age=3600")
    c.JSON(http.StatusOK, out)
}
