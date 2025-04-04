package proxy

import (
	"fmt"
	"net/http"

	"github.com/chatmcp/mcprouter/service/jsonrpc"
	"github.com/chatmcp/mcprouter/service/mcpclient"
	"github.com/chatmcp/mcprouter/service/proxy"
	"github.com/labstack/echo/v4"
)

// Messages is a handler for the messages endpoint
func Messages(c echo.Context) error {
	ctx := proxy.GetSSEContext(c)
	if ctx == nil {
		return c.String(http.StatusInternalServerError, "Failed to get SSE context")
	}

	sessionID := ctx.QueryParam("sessionid")
	if sessionID == "" {
		return ctx.JSONRPCError(jsonrpc.ErrorInvalidParams, nil)
	}

	session := ctx.GetSession(sessionID)
	if session == nil {
		return ctx.JSONRPCError(jsonrpc.ErrorInvalidParams, nil)
	}

	request, err := ctx.GetJSONRPCRequest()
	if err != nil {
		return ctx.JSONRPCError(jsonrpc.ErrorParseError, nil)
	}

	config := session.Config()
	var client *mcpclient.StdioClient

	client, err = mcpclient.GetStdioClient(config)

	if client == nil {
		return ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}

	if err != nil {
		fmt.Printf("connect to mcp server failed: %v\n", err)
		return ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}

	if err := client.Error(); err != nil {
		fmt.Printf("mcp server run failed: %v\n", err)
		return ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}
	ctx.StoreSession(sessionID, session)
	client.OnNotification(func(message []byte) {
		fmt.Printf("received notification: %s\n", message)
		session.SendMessage(string(message))
	})

	response, err := client.ForwardMessage(request)
	if err != nil {
		fmt.Printf("forward message failed: %v\n", err)
		return ctx.JSONRPCError(jsonrpc.ErrorProxyError, request.ID)
	}

	if response != nil {
		session.SendMessage(response.String())
	}

	// notification message
	return ctx.JSONRPCResponse(response)
}
