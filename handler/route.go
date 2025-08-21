package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	socks5 "go-socket5/socket5"
	"go-socket5/util"
)

// ClientPing 联通性测试
func ClientPing(c *gin.Context) {
	c.JSON(200, gin.H{"msg": "pong"})
}

// ClientAuthConfig 鉴权信息
func ClientAuthConfig(ctx *gin.Context) {
	input := socks5.Client{}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		util.ReturnJson(ctx, nil, err)
		return
	}
	ipv4, err := util.ThisIpv4Address()
	if err != nil {
		util.ReturnJson(ctx, nil, err)
		return
	}

	// 请求上来的数据尝试校验
	if err := socks5.TestConnection(input, []string{"baidu.com", "bilibili.com"}); err != nil {
		util.ReturnJson(ctx, nil, err)
	}

	localUser := uuid.New().String()
	localPasswd := uuid.New().String()

	//新增认证账户
	socks5.ServerClient.UserMap[localUser] = localPasswd

	resp := &socks5.Client{
		Host:     ipv4,
		UserName: localUser,
		Password: localPasswd,
		Port:     socks5.ServerClient.Config.Port,
	}

	util.ReturnJson(ctx, &util.RespInput{
		Data: resp,
	}, nil)
}
