package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"time"
	"vk-poster/config"
	"vk-poster/internal/controller"
	"vk-poster/internal/usecase"
	"vk-poster/internal/usecase/repo"
	"vk-poster/pkg/logging"
)

func Run(cfg config.Config) {
	hook, err := logging.NewTelegramHook(cfg.AuthToken, cfg.TargetID)
	if err != nil {
		logrus.Errorf("could not create telegram hook: %s", err)
	} else {
		logrus.AddHook(hook)
	}

	time.Local = time.FixedZone("MOSCOW", 3*60*60)

	dataSourceName := fmt.Sprintf(
		"postgresql://%s:%s@postgres:5432/%s",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)

	conn, err := sqlx.Connect("pgx", dataSourceName)
	if err != nil {
		logrus.Fatalln(err)
	}

	repository := repo.NewGroupsRepository(conn)
	useCase := usecase.NewGroupsUseCase(repository)

	if err = useCase.StartAllScanLoops(); err != nil {
		logrus.Infof("StartAllScanLoops: error occured during starting all scan loop (error: %s), restart scan loops for all groups manually", err)
	}

	router := gin.Default()

	router.LoadHTMLGlob("web/template/*.tmpl")

	controller.NewRouter(router, useCase, cfg)

	err = router.Run(fmt.Sprintf(":%d", cfg.HTTPPort))
	if err != nil {
		logrus.Fatalln("service was stopped")
	}
}
