package panlogin

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/crypto"
	"github.com/tickstep/library-go/getip"
	"github.com/tickstep/library-go/ids"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"net/url"
	"runtime"
	"strings"
	"time"
)

type LoginHelper struct {
	webHost string
}

type LoginHttpResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// QRCodeUrlResult 二维码登录页面
type QRCodeUrlResult struct {
	TokenId     string `json:"tokenId"`
	TokenUrl    string `json:"tokenUrl"`
	ExpiredTime int    `json:"expiredTime"`
}

// QRCodeLoginResult 二维码登录结果
type QRCodeLoginResult struct {
	QrCodeStatus       string `json:"qrCodeStatus"`
	SecureRefreshToken string `json:"secureRefreshToken"`
}

func NewLoginHelper(webHost string) *LoginHelper {
	return &LoginHelper{
		webHost: webHost,
	}
}

// GetQRCodeLoginUrl 获取登录二维码链接
func (h *LoginHelper) GetQRCodeLoginUrl(keyStr string) (*QRCodeUrlResult, error) {
	fullUrl := strings.Builder{}
	ipAddr, err := getip.IPInfoFromTechainBaidu()
	if err != nil {
		ipAddr = "127.0.0.1"
	}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/create?ip=%s&os=%s&arch=%s&version=%s&key=%s",
		h.webHost, ipAddr, runtime.GOOS, runtime.GOARCH, config.AppVersion, keyStr)

	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + config.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get qr code error ", err)
		return nil, err
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeUrlResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse qrcode result json error ", err2)
		return nil, err2
	}
	if r.Code != 0 {
		return nil, fmt.Errorf(r.Msg)
	}
	token := r.Data.(*QRCodeUrlResult)
	if len(token.TokenUrl) > 0 {
		u, err := url.Parse(token.TokenUrl)
		if err != nil {
			return nil, err
		}
		m, _ := url.ParseQuery(u.RawQuery)
		token.TokenId = m["tokenId"][0]
	}
	return token, nil
}

// GetQRCodeLoginResult 获取二维码登录结果
func (h *LoginHelper) GetQRCodeLoginResult(tokenId string) (*QRCodeLoginResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/result?tokenId=%s",
		h.webHost, tokenId)

	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + config.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get qr code result error ", err)
		return nil, err
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeLoginResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse qrcode result json error ", err2)
		return nil, err2
	}
	if r.Code != 0 {
		return nil, fmt.Errorf(r.Msg)
	}
	return r.Data.(*QRCodeLoginResult), nil
}

// GetRefreshToken 获取Token
func (h *LoginHelper) GetRefreshToken(tokenId string) (*QRCodeLoginResult, error) {
	fullUrl := strings.Builder{}
	fmt.Fprintf(&fullUrl, "%s/auth/tickstep/aliyunpan/token/qrcode/retrieve?tokenId=%s",
		h.webHost, tokenId)
	logger.Verboseln("do request url: " + fullUrl.String())
	header := map[string]string{
		"accept":       "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent":   "aliyunpan/" + config.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(20 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", fullUrl.String(), nil, header)
	if err != nil {
		logger.Verboseln("get refresh token result error ", err)
		return nil, err
	}

	// parse result
	r := &LoginHttpResult{}
	r.Data = &QRCodeLoginResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse refresh token result json error ", err2)
		return nil, err2
	}
	if r.Code != 0 {
		return nil, fmt.Errorf(r.Msg)
	}
	return r.Data.(*QRCodeLoginResult), nil
}

// ParseSecureRefreshToken 解密Token
func (h *LoginHelper) ParseSecureRefreshToken(keyStr, secureRefreshToken string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Verboseln("decrypt string failed, maybe the key has been changed")
		}
	}()

	if len(keyStr) == 0 {
		keyStr = ids.GetUniqueId("", 32)
	}

	if secureRefreshToken == "" {
		return "", nil
	}
	d, _ := hex.DecodeString(secureRefreshToken)

	// use the machine unique id as the key
	// but in some OS, this key will be changed if you reinstall the OS
	key := []byte(keyStr)
	if len(key) > 16 {
		key = key[:16]
	}
	r, e := crypto.DecryptAES(d, key)
	if e != nil {
		return "", nil
	}
	return string(r), nil
}
