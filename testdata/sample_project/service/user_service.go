package service

import (
	"sample_project/cache"
	"sample_project/model"
	"sample_project/repository"
)

// UserService 处理用户相关的业务逻辑
type UserService struct {
	repo  *repository.UserRepository
	cache *cache.Cache
}

// NewUserService 创建用户服务实例
func NewUserService(repo *repository.UserRepository, cache *cache.Cache) *UserService {
	return &UserService{
		repo:  repo,
		cache: cache,
	}
}

// CreateUser 创建新用户
func (s *UserService) CreateUser(name string, age int) error {
	user := model.User{
		Name: name,
		Age:  age,
	}

	if err := user.Validate(); err != nil {
		return err
	}

	return s.repo.Save(user)
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id int64) (*model.User, error) {
	// 先从缓存查找
	if user := s.cache.Get(id); user != nil {
		return user.(*model.User), nil
	}

	// 从数据库查找
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	s.cache.Set(id, user)

	return user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(user *model.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	if err := s.repo.Update(user); err != nil {
		return err
	}

	// 刷新缓存
	s.cache.Delete(user.ID)

	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id int64) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	s.cache.Delete(id)

	return nil
}
