package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"one-api/common"
	"one-api/model"
	"one-api/relay/channel/openai"
	"one-api/relay/constant"
	"one-api/relay/controller"
	"one-api/relay/util"
	"strconv"
	"strings"
)

// https://platform.openai.com/docs/api-reference/chat

func Relay(c *gin.Context) {
	relayMode := constant.RelayModeUnknown
	if strings.HasPrefix(c.Request.URL.Path, "/v1/chat/completions") {
		relayMode = constant.RelayModeChatCompletions
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/completions") {
		relayMode = constant.RelayModeCompletions
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/embeddings") {
		relayMode = constant.RelayModeEmbeddings
	} else if strings.HasSuffix(c.Request.URL.Path, "embeddings") {
		relayMode = constant.RelayModeEmbeddings
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/moderations") {
		relayMode = constant.RelayModeModerations
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/images/generations") {
		relayMode = constant.RelayModeImagesGenerations
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/edits") {
		relayMode = constant.RelayModeEdits
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/speech") {
		relayMode = constant.RelayModeAudioSpeech
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") {
		relayMode = constant.RelayModeAudioTranscription
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/translations") {
		relayMode = constant.RelayModeAudioTranslation
	}
	var err *openai.ErrorWithStatusCode
	switch relayMode {
	case constant.RelayModeImagesGenerations:
		err = controller.RelayImageHelper(c, relayMode)
	case constant.RelayModeAudioSpeech:
		fallthrough
	case constant.RelayModeAudioTranslation:
		fallthrough
	case constant.RelayModeAudioTranscription:
		err = controller.RelayAudioHelper(c, relayMode)
	default:
		err = controller.RelayTextHelper(c, relayMode)
	}
	if err != nil {
		requestId := c.GetString(common.RequestIdKey)
		retryTimesStr := c.Query("retry")
		retryTimes, _ := strconv.Atoi(retryTimesStr)
		if retryTimesStr == "" {
			retryTimes = common.RetryTimes
		}
		if retryTimes > 0 {
			c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?retry=%d", c.Request.URL.Path, retryTimes-1))
		} else {
			if err.StatusCode == http.StatusTooManyRequests {
				err.Error.Message = "网络连接遇到问题，请尝试刷新页面并重新尝试。"
			}
			err.Error.Message = common.MessageWithRequestId(err.Error.Message, requestId)
			c.JSON(err.StatusCode, gin.H{
				"error": err.Error,
			})
			//name2 := err.Error.Message
			//判断
			if relayMode == constant.RelayModeChatCompletions {
				channels, err := model.GetAllChannels(0, 0, true)
				testRequest := buildTestRequest()
				// 遍历通道列表
				for _, channel := range channels {
					// 在这里处理每个通道对象
					err, _ = testChannel(channel, *testRequest)
					// 可以根据需要处理其他字段
					if err == nil {
						if channel.Status == common.ChannelStatusManuallyDisabled {
							model.UpdateChannelStatusById(channel.Id, common.ChannelStatusEnabled)
							//common.SendSms(channel.Name, "2059654")
						}
						println("正常……", channel.Name)
					} else {
						if channel.Status == common.ChannelStatusEnabled {
							model.UpdateChannelStatusById(channel.Id, common.ChannelStatusManuallyDisabled)
							err := common.SendEmail("无法访问GPT", common.RootUserEmail, err.Error())
							common.SendSms(channel.Name, "2059401")
							if err != nil {
								println("err.Error()-------", err.Error())
							}
						}
					}
				}
			}
		}
		channelId := c.GetInt("channel_id")
		common.LogError(c.Request.Context(), fmt.Sprintf("relay error (channel #%d): %s", channelId, err.Message))
		// https://platform.openai.com/docs/guides/error-codes/api-errors
		if util.ShouldDisableChannel(&err.Error, err.StatusCode) {
			channelId := c.GetInt("channel_id")
			channelName := c.GetString("channel_name")
			disableChannel(channelId, channelName, err.Message)
		}
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := openai.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := openai.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
