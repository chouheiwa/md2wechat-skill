package image

import (
	"fmt"
	"os"

	"github.com/geekjourneyx/md2wechat-skill/internal/config"
	"github.com/geekjourneyx/md2wechat-skill/internal/wechat"
	"go.uber.org/zap"
)

// Processor 图片处理器
type Processor struct {
	cfg        *config.Config
	log        *zap.Logger
	ws         *wechat.Service
	compressor *Compressor
}

// NewProcessor 创建图片处理器
func NewProcessor(cfg *config.Config, log *zap.Logger) *Processor {
	return &Processor{
		cfg:        cfg,
		log:        log,
		ws:         wechat.NewService(cfg, log),
		compressor: NewCompressor(log, cfg.MaxImageWidth, cfg.MaxImageSize),
	}
}

// UploadResult 上传结果
type UploadResult struct {
	MediaID   string `json:"media_id"`
	WechatURL string `json:"wechat_url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// UploadLocalImage 上传本地图片
func (p *Processor) UploadLocalImage(filePath string) (*UploadResult, error) {
	p.log.Info("uploading local image", zap.String("path", filePath))

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 检查图片格式
	if !IsValidImageFormat(filePath) {
		return nil, fmt.Errorf("unsupported image format: %s", filePath)
	}

	// 如果需要压缩，先处理
	processedPath := filePath
	if p.cfg.CompressImages {
		compressedPath, compressed, err := p.compressor.CompressImage(filePath)
		if err != nil {
			p.log.Warn("compress failed, using original", zap.Error(err))
		} else if compressed {
			processedPath = compressedPath
			defer os.Remove(compressedPath)
			p.log.Info("using compressed image", zap.String("path", processedPath))
		}
	}

	// 上传到微信
	result, err := p.ws.UploadMaterialWithRetry(processedPath, 3)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		MediaID:   result.MediaID,
		WechatURL: result.WechatURL,
	}, nil
}

// DownloadAndUpload 下载在线图片并上传
func (p *Processor) DownloadAndUpload(url string) (*UploadResult, error) {
	p.log.Info("downloading and uploading image", zap.String("url", url))

	// 下载图片
	tmpPath, err := wechat.DownloadFile(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpPath)

	// 检查格式
	if !IsValidImageFormat(tmpPath) {
		return nil, fmt.Errorf("downloaded file is not a valid image")
	}

	// 压缩（如果需要）
	processedPath := tmpPath
	if p.cfg.CompressImages {
		compressedPath, compressed, err := p.compressor.CompressImage(tmpPath)
		if err != nil {
			p.log.Warn("compress failed, using original", zap.Error(err))
		} else if compressed {
			processedPath = compressedPath
			defer os.Remove(compressedPath)
			p.log.Info("using compressed image", zap.String("path", processedPath))
		}
	}

	// 上传到微信
	result, err := p.ws.UploadMaterialWithRetry(processedPath, 3)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		MediaID:   result.MediaID,
		WechatURL: result.WechatURL,
	}, nil
}

// GetImageInfo 获取图片信息
func (p *Processor) GetImageInfo(filePath string) (*ImageInfo, error) {
	return GetImageInfo(filePath)
}

// CompressImage 压缩图片（公开方法）
func (p *Processor) CompressImage(filePath string) (string, bool, error) {
	return p.compressor.CompressImage(filePath)
}

// SetCompressQuality 设置压缩质量
func (p *Processor) SetCompressQuality(quality int) {
	p.compressor.SetQuality(quality)
}
