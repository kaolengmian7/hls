// Package code 负责 6 位数字房间 Code 的生成与全局唯一性管理。
//
// Generator 内部持有已分配 Code 的集合，对外提供：
//   - Allocate(): 生成一个未占用的 Code。
//   - Release(code): 释放一个 Code 以便重用。
//
// 实现：crypto/rand 保证不可预测；首位非零；最大重试次数 32。
package code

import (
	"crypto/rand"
	"encoding/binary"
	"sync"

	"hls.dev/server/pkg/errors"
)

const (
	// CodeLength 房间 Code 长度。
	CodeLength = 6
	// MaxRetry 最大重试次数。极端碰撞下抛出错误。
	MaxRetry = 32
)

// Generator 是房间 Code 的全局分配器。
type Generator struct {
	mu   sync.Mutex
	used map[string]struct{}
}

// NewGenerator 创建一个新的 Generator。
func NewGenerator() *Generator {
	return &Generator{used: make(map[string]struct{}, 256)}
}

// Allocate 申请一个未占用的 6 位 Code。
func (g *Generator) Allocate() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := 0; i < MaxRetry; i++ {
		c, err := randomCode()
		if err != nil {
			return "", errors.WithMessage(err, "random code")
		}
		if _, ok := g.used[c]; ok {
			continue
		}
		g.used[c] = struct{}{}
		return c, nil
	}
	return "", errors.New(errors.CodeInternal, "code pool exhausted after %d retries", MaxRetry)
}

// Release 释放一个 Code，使其可被重新分配。
func (g *Generator) Release(c string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.used, c)
}

// Has 返回 Code 是否已被占用（仅诊断用）。
func (g *Generator) Has(c string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.used[c]
	return ok
}

// IsValid 校验输入是否为合法 6 位数字 Code。
func IsValid(c string) bool {
	if len(c) != CodeLength {
		return false
	}
	for i := 0; i < CodeLength; i++ {
		if c[i] < '0' || c[i] > '9' {
			return false
		}
	}
	return true
}

// randomCode 生成首位非零的 6 位数字。
func randomCode() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint64(buf[:])
	// 首位 1-9，后五位 0-9
	first := byte('1' + n%9)
	rest := n / 9
	out := make([]byte, CodeLength)
	out[0] = first
	for i := 1; i < CodeLength; i++ {
		out[i] = byte('0' + rest%10)
		rest /= 10
	}
	return string(out), nil
}
