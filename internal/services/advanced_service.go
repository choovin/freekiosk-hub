// Copyright (C) 2026 wared2003
// SPDX-License-Identifier: AGPL-3.0-or-later
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// AdvancedService 高级功能服务接口
type AdvancedService interface {
	// DevicePhoto 设备拍照
	CreateDevicePhoto(tenantID, deviceID, url, thumbnailURL string, fileSize int64, width, height int, description string) (*models.DevicePhoto, error)
	GetDevicePhoto(id string) (*models.DevicePhoto, error)
	ListDevicePhotos(tenantID, deviceID string, limit, offset int) ([]*models.DevicePhoto, int64, error)
	DeleteDevicePhoto(id string) error

	// Contact 联系人
	CreateContact(tenantID, deviceID, name, phone, email, company, jobTitle, address, note string) (*models.Contact, error)
	GetContact(id string) (*models.Contact, error)
	UpdateContact(contact *models.Contact) error
	DeleteContact(id string) error
	ListContacts(tenantID, deviceID string, limit, offset int) ([]*models.Contact, int64, error)
	SearchContacts(tenantID, keyword string, limit, offset int) ([]*models.Contact, int64, error)

	// LDAP配置
	CreateLDAPConfig(tenantID, name, server string, port int, useSSL bool, baseDN, bindDN, bindPassword, userFilter, groupFilter string, syncInterval int) (*models.LDAPConfig, error)
	GetLDAPConfig(id string) (*models.LDAPConfig, error)
	GetLDAPConfigByTenant(tenantID string) (*models.LDAPConfig, error)
	UpdateLDAPConfig(config *models.LDAPConfig) error
	DeleteLDAPConfig(id string) error
	ListLDAPConfigs(tenantID string) ([]*models.LDAPConfig, error)
	TestLDAPConnection(config *models.LDAPConfig) (bool, string, error)
	SyncLDAPUsers(configID string) ([]*models.LDAPUser, error)
	ListLDAPUsers(tenantID string, limit, offset int) ([]*models.LDAPUser, int64, error)

	// WhiteLabel配置
	CreateWhiteLabelConfig(tenantID, name string, config *WhiteLabelInput) (*models.WhiteLabelConfig, error)
	GetWhiteLabelConfig(id string) (*models.WhiteLabelConfig, error)
	GetWhiteLabelConfigByTenant(tenantID string) (*models.WhiteLabelConfig, error)
	UpdateWhiteLabelConfig(config *models.WhiteLabelConfig) error
	DeleteWhiteLabelConfig(id string) error
}

// WhiteLabelInput 白标配置输入
type WhiteLabelInput struct {
	LogoURL          string
	FaviconURL       string
	PrimaryColor    string
	SecondaryColor  string
	AccentColor     string
	BackgroundColor string
	TextColor       string
	CustomCSS       string
	CustomJS        string
	FooterText      string
	LoginBgURL      string
	Enabled         bool
}

// DefaultAdvancedService 默认实现
type DefaultAdvancedService struct {
	repo repositories.AdvancedRepository
}

// NewAdvancedService 创建高级功能服务
func NewAdvancedService(repo repositories.AdvancedRepository) *DefaultAdvancedService {
	return &DefaultAdvancedService{repo: repo}
}

// DevicePhoto Methods

func (s *DefaultAdvancedService) CreateDevicePhoto(tenantID, deviceID, url, thumbnailURL string, fileSize int64, width, height int, description string) (*models.DevicePhoto, error) {
	photo := &models.DevicePhoto{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		DeviceID:     deviceID,
		URL:          url,
		ThumbnailURL: thumbnailURL,
		FileSize:     fileSize,
		Width:        width,
		Height:       height,
		CapturedAt:   time.Now(),
		Description:   description,
	}

	if err := s.repo.CreateDevicePhoto(photo); err != nil {
		return nil, err
	}

	return photo, nil
}

func (s *DefaultAdvancedService) GetDevicePhoto(id string) (*models.DevicePhoto, error) {
	return s.repo.GetDevicePhoto(id)
}

func (s *DefaultAdvancedService) ListDevicePhotos(tenantID, deviceID string, limit, offset int) ([]*models.DevicePhoto, int64, error) {
	return s.repo.ListDevicePhotos(tenantID, deviceID, limit, offset)
}

func (s *DefaultAdvancedService) DeleteDevicePhoto(id string) error {
	return s.repo.DeleteDevicePhoto(id)
}

// Contact Methods

func (s *DefaultAdvancedService) CreateContact(tenantID, deviceID, name, phone, email, company, jobTitle, address, note string) (*models.Contact, error) {
	now := time.Now().Unix()
	contact := &models.Contact{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		DeviceID:    deviceID,
		Name:        name,
		Phone:       phone,
		Email:       email,
		Company:     company,
		JobTitle:    jobTitle,
		Address:     address,
		Note:        note,
		Starred:     false,
		Frequency:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateContact(contact); err != nil {
		return nil, err
	}

	return contact, nil
}

func (s *DefaultAdvancedService) GetContact(id string) (*models.Contact, error) {
	return s.repo.GetContact(id)
}

func (s *DefaultAdvancedService) UpdateContact(contact *models.Contact) error {
	return s.repo.UpdateContact(contact)
}

func (s *DefaultAdvancedService) DeleteContact(id string) error {
	return s.repo.DeleteContact(id)
}

func (s *DefaultAdvancedService) ListContacts(tenantID, deviceID string, limit, offset int) ([]*models.Contact, int64, error) {
	return s.repo.ListContacts(tenantID, deviceID, limit, offset)
}

func (s *DefaultAdvancedService) SearchContacts(tenantID, keyword string, limit, offset int) ([]*models.Contact, int64, error) {
	return s.repo.SearchContacts(tenantID, keyword, limit, offset)
}

// LDAPConfig Methods

func (s *DefaultAdvancedService) CreateLDAPConfig(tenantID, name, server string, port int, useSSL bool, baseDN, bindDN, bindPassword, userFilter, groupFilter string, syncInterval int) (*models.LDAPConfig, error) {
	if port <= 0 {
		port = 389
	}
	if syncInterval <= 0 {
		syncInterval = 60
	}

	now := time.Now().Unix()
	config := &models.LDAPConfig{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Name:         name,
		Server:       server,
		Port:         port,
		UseSSL:       useSSL,
		BaseDN:       baseDN,
		BindDN:       bindDN,
		BindPassword: bindPassword,
		UserFilter:   userFilter,
		GroupFilter:  groupFilter,
		SyncInterval: syncInterval,
		Enabled:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateLDAPConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (s *DefaultAdvancedService) GetLDAPConfig(id string) (*models.LDAPConfig, error) {
	return s.repo.GetLDAPConfig(id)
}

func (s *DefaultAdvancedService) GetLDAPConfigByTenant(tenantID string) (*models.LDAPConfig, error) {
	return s.repo.GetLDAPConfigByTenant(tenantID)
}

func (s *DefaultAdvancedService) UpdateLDAPConfig(config *models.LDAPConfig) error {
	return s.repo.UpdateLDAPConfig(config)
}

func (s *DefaultAdvancedService) DeleteLDAPConfig(id string) error {
	return s.repo.DeleteLDAPConfig(id)
}

func (s *DefaultAdvancedService) ListLDAPConfigs(tenantID string) ([]*models.LDAPConfig, error) {
	return s.repo.ListLDAPConfigs(tenantID)
}

func (s *DefaultAdvancedService) TestLDAPConnection(config *models.LDAPConfig) (bool, string, error) {
	return s.repo.TestLDAPConnection(config)
}

func (s *DefaultAdvancedService) SyncLDAPUsers(configID string) ([]*models.LDAPUser, error) {
	return s.repo.SyncLDAPUsers(configID)
}

func (s *DefaultAdvancedService) ListLDAPUsers(tenantID string, limit, offset int) ([]*models.LDAPUser, int64, error) {
	return s.repo.ListLDAPUsers(tenantID, limit, offset)
}

// WhiteLabelConfig Methods

func (s *DefaultAdvancedService) CreateWhiteLabelConfig(tenantID, name string, input *WhiteLabelInput) (*models.WhiteLabelConfig, error) {
	now := time.Now().Unix()
	config := &models.WhiteLabelConfig{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            name,
		LogoURL:         input.LogoURL,
		FaviconURL:      input.FaviconURL,
		PrimaryColor:    input.PrimaryColor,
		SecondaryColor: input.SecondaryColor,
		AccentColor:     input.AccentColor,
		BackgroundColor: input.BackgroundColor,
		TextColor:      input.TextColor,
		CustomCSS:      input.CustomCSS,
		CustomJS:       input.CustomJS,
		FooterText:     input.FooterText,
		LoginBgURL:     input.LoginBgURL,
		Enabled:        input.Enabled,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Set defaults if empty
	if config.PrimaryColor == "" {
		config.PrimaryColor = "#1976D2"
	}
	if config.SecondaryColor == "" {
		config.SecondaryColor = "#424242"
	}
	if config.AccentColor == "" {
		config.AccentColor = "#FF5722"
	}
	if config.BackgroundColor == "" {
		config.BackgroundColor = "#FFFFFF"
	}
	if config.TextColor == "" {
		config.TextColor = "#212121"
	}

	if err := s.repo.CreateWhiteLabelConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (s *DefaultAdvancedService) GetWhiteLabelConfig(id string) (*models.WhiteLabelConfig, error) {
	return s.repo.GetWhiteLabelConfig(id)
}

func (s *DefaultAdvancedService) GetWhiteLabelConfigByTenant(tenantID string) (*models.WhiteLabelConfig, error) {
	return s.repo.GetWhiteLabelConfigByTenant(tenantID)
}

func (s *DefaultAdvancedService) UpdateWhiteLabelConfig(config *models.WhiteLabelConfig) error {
	return s.repo.UpdateWhiteLabelConfig(config)
}

func (s *DefaultAdvancedService) DeleteWhiteLabelConfig(id string) error {
	return s.repo.DeleteWhiteLabelConfig(id)
}

// InitAdvancedSchema 初始化高级功能表结构
func InitAdvancedSchema(ctx context.Context, repo repositories.AdvancedRepository) error {
	return repo.InitSchema(ctx)
}
