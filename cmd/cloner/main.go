package main

import (
	"os"

	"github.com/MrTomSawyer/k8s-image-controller/internal/cloner"
	conf "github.com/MrTomSawyer/k8s-image-controller/internal/config"
	"github.com/MrTomSawyer/k8s-image-controller/internal/controller"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctx := ctrl.SetupSignalHandler()
	logger := ctrl.Log.WithName("Init")

	logger.Info("====================================")
	logger.Info("*** Image clone controller setup ***")
	logger.Info("====================================")

	logger.Info("creating a manager...")
	mgr, err := ctrl.NewManager(config.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		logger.Error(err, "failed to create manager")
		os.Exit(1)
	}

	logger.Info("initializing config...")
	cfg := conf.New(mgr.GetAPIReader()).ParseFlags()
	err = cfg.RetrieveDockerSecrets(ctx, cfg.ControllerParams.Namespace, cfg.ControllerParams.DockerSecretName)
	if err != nil {
		logger.Error(err, "failed retrieve docker secrets")
		os.Exit(1)
	}

	logger.Info("creating an image cloner...")
	imageCloner := cloner.NewImageCloner(mgr.GetClient(), mgr.GetAPIReader(), ctrl.Log.WithName("Image Cloner"))
	err = imageCloner.SetupForDocker(cfg.DockerConfig.Username, cfg.DockerConfig.Password)
	if err != nil {
		logger.Error(err, "failed to create an image cloner")
		os.Exit(1)
	}

	logger.Info("creating a deployment reconciler...")
	err = controller.NewDeploymentReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		imageCloner,
		ctrl.Log.WithName("Deployment reconciler"),
		cfg.DockerConfig.Username,
	).Register(mgr)
	if err != nil {
		logger.Error(err, "failed to register deployment reconciler")
		os.Exit(1)
	}

	logger.Info("creating a daemonset reconciler...")
	err = controller.NewDaemonsetReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		imageCloner,
		ctrl.Log.WithName("Daemonset reconciler"),
		cfg.DockerConfig.Username,
	).Register(mgr)
	if err != nil {
		logger.Error(err, "failed to register daemonset reconciler")
		os.Exit(1)
	}

	logger.Info("starting a manager...")
	if err := mgr.Start(ctx); err != nil {
		logger.Error(err, "failed start a manager")
		os.Exit(1)
	}
}
