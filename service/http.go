package service

import (
	"strconv"

	"github.com/EducationEKT/EKT/log"
	"github.com/gin-gonic/gin"
)

type HTTPServer struct {
	Engine *gin.Engine
	Port   int
}

func NewHTTPServer(port int) *HTTPServer {
	return &HTTPServer{
		Engine: gin.Default(),
		Port:   port,
	}
}

func (server HTTPServer) RegistFunctions(group FuncGroup) {
	module := server.Engine.Group(group.GroupName)
	for _, function := range group.Functions {
		module.Any("api/"+function.FuncName, func(context *gin.Context) {
			params := make([]interface{}, 0)
			for _, param := range function.Params {
				switch param.Type {
				case PARAM_TYPE_BODY:
					body, err := context.GetRawData()
					if err != nil {
						log.Crit("Invalid function")
					}
					params = append(params, body)
				case PARAM_TYPE_INT:
					params = append(params, context.GetInt(param.Name))
				case PARAM_TYPE_FLOAT64:
					params = append(params, context.GetFloat64(param.Name))
				case PARAM_TYPE_STRING:
					params = append(params, context.GetString(param.Name))
					if param.From == PARAM_FROM_QUERY {
						params = append(params, context.Query(param.Name))
					}
				}
			}
			function.Func(params...)
		})
	}
}

func (server HTTPServer) RegistGroups(groups []FuncGroup) {
	for _, group := range groups {
		server.RegistFunctions(group)
	}
}

func (server HTTPServer) Start() error {
	if err := server.Engine.Run(":" + strconv.Itoa(server.Port)); err != nil {
		return err
	}
	return nil
}

func (server HTTPServer) Stop() error {
	return nil
}
