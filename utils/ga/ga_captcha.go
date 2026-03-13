package ga

import (
	"image/color"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type CaptchaResult struct {
	Id          string `json:"id"`
	Show        bool   `json:"show"`
	Base64Blog  string `json:"img"`
	VerifyValue string `json:"code"`
	ExpireTime  int64  `json:"expireTime"`
}

var Expiration = 3 * time.Minute

var redisStore *RedisStore

type RedisStore struct {
	redis *redis.Redis
}

func NewRedisStore(rds *redis.Redis) *RedisStore {
	return &RedisStore{redis: rds}
}

func (s *RedisStore) Set(id string, value string) error {
	key := "captcha:" + id
	return s.redis.Setex(key, value, int(Expiration.Seconds()))
}

func (s *RedisStore) Get(id string, clear bool) string {
	key := "captcha:" + id
	value, err := s.redis.Get(key)
	if err != nil {
		return ""
	}
	if clear {
		s.redis.Del(key)
	}
	return value
}

func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	key := "captcha:" + id
	value, err := s.redis.Get(key)
	if err != nil {
		return false
	}
	if clear {
		s.redis.Del(key)
	}
	return value == answer
}

func InitCaptchaStore(rds *redis.Redis) {
	redisStore = NewRedisStore(rds)
}

func GenerateCaptcha(show bool) (interface{}, error) {
	driver := base64Captcha.NewDriverMath(39, 110, 0, 0, &color.RGBA{0, 0, 0, 1}, nil, []string{"wqy-microhei.ttc"})

	var captcha *base64Captcha.Captcha
	if redisStore != nil {
		captcha = base64Captcha.NewCaptcha(driver, redisStore)
	} else {
		store := base64Captcha.NewMemoryStore(10240, Expiration)
		captcha = base64Captcha.NewCaptcha(driver, store)
	}

	id, b64s, answer, err := captcha.Generate()
	if err != nil {
		return "", err
	}

	if redisStore != nil {
		redisStore.Set(id, answer)
	}

	captchaResult := CaptchaResult{
		Id:          id,
		Show:        show,
		Base64Blog:  b64s,
		VerifyValue: answer,
		ExpireTime:  time.Now().Add(Expiration).UnixMilli(),
	}
	return captchaResult, nil
}

func VerifyCaptcha(id string, value string) bool {
	if redisStore != nil {
		return redisStore.Verify(id, value, true)
	}

	return false
}
