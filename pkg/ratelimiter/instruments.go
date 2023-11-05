package main

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/time/rate"
)

var ErrLimitExceed = errors.New("Rate limit exceed!")

// // 该限流器是基于令牌桶算法实现的
// 新建一个限流中间件
func NewTokenBucketLimitterWithBuildIn(bkt *rate.Limiter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if !bkt.Allow() {
				return nil, ErrLimitExceed
			}
			return next(ctx, request)
		}
	}
}

// 创建一个请求频率限制的中间件。
// 它接受两个参数：interval表示间隔时间，以秒为单位；burst表示可以在间隔时间内同时处理的最大请求数量。
func DynamicLImitter(interval int, burst int) endpoint.Middleware {
	bucket := rate.NewLimiter(rate.Every(time.Second*time.Duration(interval)), burst)
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if !bucket.Allow() {
				return nil, ErrLimitExceed
			}
			return next(ctx, request)
		}
	}
}
