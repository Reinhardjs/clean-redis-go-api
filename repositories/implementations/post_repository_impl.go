package implementations

import (
	"dot-crud-redis-go-api/models"
	"dot-crud-redis-go-api/repositories"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/gorm"
)

type PostRepoImpl struct {
	DB          *gorm.DB
	RedisClient redis.Conn
}

func CreatePostRepo(DB *gorm.DB, RedisClient redis.Conn) repositories.PostRepo {
	return &PostRepoImpl{DB, RedisClient}
}

func (e *PostRepoImpl) Create(post *models.Post) (*models.Post, error) {
	result := e.DB.Model(&models.Post{}).Create(post)

	if result.Error != nil {
		return &models.Post{}, fmt.Errorf("DB error : %v", result.Error)
	}

	return post, nil
}

func (e *PostRepoImpl) ReadAll() (*[]models.Post, error) {
	posts := make([]models.Post, 0)
	err := e.DB.Table("posts").Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("DB error : %v", err)
	}

	return &posts, nil
}

func (e *PostRepoImpl) ReadById(id int) (*models.Post, error) {
	post := &models.Post{}

	// Get JSON blob from Redis
	redisResult, err := e.RedisClient.Do("GET", "post:"+strconv.Itoa(id))

	if err != nil {
		// Failed getting data from redis
		return nil, err
	}

	if redisResult == nil {

		errorRead := e.DB.Table("posts").Where("id = ?", id).First(post).Error

		if errorRead != nil {
			return nil, errorRead
		}

		postJSON, err := json.Marshal(post)
		if err != nil {
			return nil, err
		}

		// Save JSON blob to Redis
		_, saveRedisError := e.RedisClient.Do("SET", "post:"+strconv.Itoa(id), postJSON)

		if saveRedisError != nil {
			// Failed saving data to redis
			return nil, saveRedisError
		}
	} else {
		json.Unmarshal(redisResult.([]byte), &post)
	}

	return post, nil
}

func (e *PostRepoImpl) Update(id int, post *models.Post) (*models.Post, error) {
	updatedPost := &models.Post{}
	result := e.DB.Model(updatedPost).Where("id = ?", id).Updates(models.Post{Title: post.Title, Description: post.Description})

	if result.Error != nil {
		return nil, fmt.Errorf("DB error : %v", result.Error)
	}

	// Delete JSON blob from Redis
	_, redisDeleteErr := e.RedisClient.Do("DEL", "post:"+strconv.Itoa(id))

	if redisDeleteErr != nil {
		// Failed deleting data from redis
		return nil, redisDeleteErr
	}

	updatedPost.ID = id

	return updatedPost, nil
}

func (e *PostRepoImpl) Delete(id int) (map[string]interface{}, error) {
	result := e.DB.Delete(&models.Post{}, id)

	// Delete JSON blob from Redis
	_, redisDeleteErr := e.RedisClient.Do("DEL", "post:"+strconv.Itoa(id))

	if redisDeleteErr != nil {
		// Failed deleting data from redis
		return nil, redisDeleteErr
	}

	return map[string]interface{}{
		"rows_affected": result.RowsAffected,
	}, nil
}
