package routes

import (
	"rumosaudavel-api/internal/handlers"
	"rumosaudavel-api/internal/middleware"

	"github.com/joho/godotenv"

	"github.com/labstack/echo/v4"
	"database/sql"
)

func init() {
	godotenv.Load()
}

func Routes(e *echo.Echo, db *sql.DB) {
	api := e.Group("/rumosaudavel-api")

	userHandler := handlers.NewUserHandler(db)
	
	api.POST("/login", userHandler.Login)
	api.POST("/usuarios", userHandler.Criar)
	api.GET("/usuarios", userHandler.Lista)
	api.DELETE("/usuarios/:id", userHandler.Apagar)
	api.POST("/esqueci", userHandler.Esqueci)
	api.POST("/reset/:token", userHandler.ResetSenha)
	api.POST("/logout", userHandler.Logout)

	dashboardHandler := handlers.NewDashboardHandler(db)

	api.GET("/dashboard/home", dashboardHandler.Home)
	api.GET("/dashboard/bugs", dashboardHandler.Bugs)

	participanteHandler := handlers.NewParticipanteHandler(db)

	api.POST("/participantes", participanteHandler.Criar)
	api.GET("/participantes/:id", participanteHandler.Lista)
	api.GET("/participantes/termometro/:id", participanteHandler.Termometro)
	api.GET("/participantes/termometro-cor", participanteHandler.TermometroCor)
	api.DELETE("/participantes/:id", participanteHandler.Apagar)
	api.DELETE("/participantes/empresa/:id", participanteHandler.ApagarPorEmpresa)
	api.POST("/participantes/gera-indices", participanteHandler.GeraIndices)
	api.POST("/participantes/upload", participanteHandler.UploadExcel)

	programaHandler := handlers.NewProgramaHandler(db)

	api.POST("/programas", programaHandler.Criar)
	api.PUT("/programas", programaHandler.Editar)
	api.GET("/programas", programaHandler.Lista)
	api.POST("/programas/vincular", programaHandler.VincularEmpresa)
	api.POST("/programas/intervalo", programaHandler.DefinirIntervalo)
	api.DELETE("/programas/intervalo/:empresa/:programa", programaHandler.ResetarIntervalo)
	api.DELETE("/programas/:id", programaHandler.Apagar)
	api.POST("/programas/:id/duplicar", programaHandler.Duplicar)

	questionarioHandler := handlers.NewQuestionarioHandler(db)
	
	api.GET("/questionarios", questionarioHandler.Listar)
	api.POST("/questionarios", questionarioHandler.Criar)
	api.GET("/questionarios/:id", questionarioHandler.Info)
	api.PUT("/questionarios/:id", questionarioHandler.Editar)
	api.DELETE("/questionarios/:id", questionarioHandler.Apagar)
	api.POST("/questionarios/:id/duplicar", questionarioHandler.Duplicar)

	api.POST("/questionarios/:id/perguntas", questionarioHandler.AddPergunta)
	api.PUT("/perguntas/:id", questionarioHandler.EditarPergunta)
	api.DELETE("/perguntas/:id", questionarioHandler.ApagarPergunta)

	api.POST("/perguntas/:id/alternativas", questionarioHandler.AddAlternativa)
	api.DELETE("/alternativas/:id", questionarioHandler.ApagarAlternativa)
	api.POST("/questionarios/:id/ordenacao", questionarioHandler.SalvarOrdem)

	api.POST("/questionarios/:id/intervalos", questionarioHandler.SalvarIntervalo)
	api.DELETE("/intervalos/:id", questionarioHandler.ApagarIntervalo)

	api.POST("/questionarios/:id/restaurar", questionarioHandler.Restaurar)
	api.POST("/questionarios/:id/delete-definitivo", questionarioHandler.DeleteDefinitivo)

	questionarioInteracaoHandler := handlers.NewQuestionarioInteracaoHandler(db)

	api.GET("/participante/home", questionarioInteracaoHandler.Home)
	api.GET("/participante/questionario", questionarioInteracaoHandler.Questionario)
	api.POST("/participante/responder", questionarioInteracaoHandler.ProcessaResposta)
	api.GET("/participante/relatorio", questionarioInteracaoHandler.Relatorio)
	api.GET("/participante/prontuario", questionarioInteracaoHandler.Prontuario)
	api.GET("/participante/contato", questionarioInteracaoHandler.Contato)

	protected := api.Group("")
	protected.Use(middleware.JWTMiddleware)
}
