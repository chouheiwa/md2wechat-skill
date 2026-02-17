// Package converter 提供 Markdown 到微信公众号 HTML 的转换功能
// 通过 AI 模式（Claude 生成）将 Markdown 转换为微信公众号格式 HTML
package converter

import (
	"regexp"
	"strings"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"go.uber.org/zap"
)

// ImageType 图片类型
type ImageType string

const (
	ImageTypeLocal  ImageType = "local"  // 本地图片
	ImageTypeOnline ImageType = "online" // 在线图片
)

// ConvertRequest 转换请求
type ConvertRequest struct {
	// 基础输入
	Markdown string // Markdown 内容
	Theme    string // 主题名称 / AI 提示词名称

	// AI 模式专用
	CustomPrompt string // 自定义提示词
}

// ImageRef 图片引用
type ImageRef struct {
	Index       int       // 位置索引
	Original    string    // 原始路径
	Placeholder string    // HTML 中的占位符 <!-- IMG:0 -->
	WechatURL   string    // 上传后的 URL (处理完成后)
	Type        ImageType // 图片类型
}

// ConvertResult 转换结果
type ConvertResult struct {
	HTML    string     // 生成的 HTML（含占位符）
	Theme   string     // 使用的主题
	Images  []ImageRef // 图片引用列表
	Success bool       // 是否成功
	Error   string     // 错误信息
}

// Converter 转换器接口
type Converter interface {
	// Convert 执行转换
	Convert(req *ConvertRequest) *ConvertResult

	// ExtractImages 从 Markdown 中提取图片引用
	ExtractImages(markdown string) []ImageRef
}

// converter 转换器实现
type converter struct {
	cfg           *config.Config
	log           *zap.Logger
	theme         *ThemeManager
	promptBuilder *PromptBuilder
}

// NewConverter 创建转换器
func NewConverter(cfg *config.Config, log *zap.Logger) Converter {
	return &converter{
		cfg:           cfg,
		log:           log,
		theme:         NewThemeManager(),
		promptBuilder: NewPromptBuilder(),
	}
}

// Convert 执行转换
func (c *converter) Convert(req *ConvertRequest) *ConvertResult {
	result := &ConvertResult{
		Theme: req.Theme,
	}

	// 验证请求
	if err := c.validateRequest(req); err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	return c.convertViaAI(req)
}

// validateRequest 验证请求参数
func (c *converter) validateRequest(req *ConvertRequest) error {
	if req.Markdown == "" {
		return ErrEmptyMarkdown
	}

	if req.Theme == "" {
		req.Theme = "autumn-warm"
	}

	return nil
}

// ExtractImages 从 Markdown 中提取图片引用
func (c *converter) ExtractImages(markdown string) []ImageRef {
	var images []ImageRef

	// 匹配本地图片: ![alt](./path/to/image.png)
	localPattern := regexp.MustCompile(`!\[([^\]]*)\]\((\.\/[^)]+)\)`)
	for i, match := range localPattern.FindAllStringSubmatch(markdown, -1) {
		if len(match) >= 3 {
			images = append(images, ImageRef{
				Index:       i,
				Original:    match[2],
				Placeholder: "",
				Type:        ImageTypeLocal,
			})
		}
	}

	// 匹配在线图片: ![alt](https://...)
	onlinePattern := regexp.MustCompile(`!\[([^\]]*)\]\((https?://[^)]+)\)`)
	offset := len(images)
	for i, match := range onlinePattern.FindAllStringSubmatch(markdown, -1) {
		if len(match) >= 3 {
			images = append(images, ImageRef{
				Index:       offset + i,
				Original:    match[2],
				Placeholder: "",
				Type:        ImageTypeOnline,
			})
		}
	}

	return images
}

// ReplaceImagePlaceholders 在 HTML 中替换图片占位符
func ReplaceImagePlaceholders(html string, images []ImageRef) string {
	result := html
	for _, img := range images {
		if img.WechatURL != "" {
			// 替换占位符为实际图片标签
			imgTag := `<img src="` + img.WechatURL + `" style="max-width:100%;height:auto;display:block;margin:20px auto;" />`
			result = strings.ReplaceAll(result, img.Placeholder, imgTag)
		}
	}
	return result
}

// InsertImagePlaceholders 在 HTML 中插入图片占位符
func InsertImagePlaceholders(html string, images []ImageRef) string {
	// 简化实现：直接返回原 HTML
	// 实际的占位符插入由 AI 或 API 转换时处理
	return html
}

// 错误定义
var (
	ErrEmptyMarkdown = &ConvertError{Code: "EMPTY_MARKDOWN", Message: "markdown content cannot be empty"}
	ErrInvalidTheme  = &ConvertError{Code: "INVALID_THEME", Message: "invalid theme name"}
	ErrAIFailure     = &ConvertError{Code: "AI_FAILURE", Message: "AI generation failed"}
)

// ConvertError 转换错误
type ConvertError struct {
	Code    string
	Message string
	Err     error
}

func (e *ConvertError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *ConvertError) Unwrap() error {
	return e.Err
}

// GetPromptBuilder 获取 Prompt 构建器（用于外部访问）
func GetPromptBuilder() *PromptBuilder {
	return NewPromptBuilder()
}

// ValidateAIRequest 验证 AI 转换请求
func ValidateAIRequest(prompt string) *ValidationResult {
	return ValidatePromptContent(prompt)
}

// GetMarkdownTitle 提取 Markdown 标题
func GetMarkdownTitle(markdown string) string {
	return ParseMarkdownTitle(markdown)
}

// EstimateTokens 估算文本 token 数量
func EstimateTokens(text string) int {
	return EstimateTokenCount(text)
}
