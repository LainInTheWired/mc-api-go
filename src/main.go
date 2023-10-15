package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const (
	gameServerImage      = "GAMESERVER_IMAGE"
	isHelmTest           = "IS_HELM_TEST"
	gameserversNamespace = "default"
	defaultNs            = "default"
)

func main() {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "healthy",
		})
	})
	r.GET("/create", createMcServer)
	// r.GET("/pvc", createPVC)
	r.Run()
}

func createPVC(kubernetesClient kubernetes.Interface, namespace string, pvcName string, storageClassName string, storageSize string) (*corev1.PersistentVolumeClaim, error) {

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}
	createdPVC, err := kubernetesClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdPVC, nil
}

func createMcServer(c *gin.Context) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	kubernetesClient, err := kubernetes.NewForConfig(config)

	logger := runtime.NewLoggerWithSource("main")
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	createPVC, err := createPVC(kubernetesClient, "default", "test-pvc-serial", "longhorn", "10Gi")
	if err != nil {
		panic(err)
	}
	fmt.Print(createPVC)
	// Create a GameServer
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "helm-test-server-",
			// Namespace:    viper.GetString(gameserversNamespace),
			Namespace: "default",
		},
		Spec: agonesv1.GameServerSpec{
			Container: "simple-game-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 80,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "simple-game-server",
							// Image: viper.GetString(gameServerImage),
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	fmt.Println(gs)
	newGS, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatal("Unable to create GameServer: %v", err)
	}
	logrus.Infof("New GameServer name is: %s", newGS.ObjectMeta.Name)

	if viper.GetBool(isHelmTest) {
		err = wait.PollImmediate(1*time.Second, 60*time.Second, func() (bool, error) {
			checkGs, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Get(ctx, newGS.Name, metav1.GetOptions{})

			if err != nil {
				logrus.WithError(err).Warn("error retrieving gameserver")
				return false, nil
			}

			state := agonesv1.GameServerStateReady
			logger.WithField("gs", checkGs.ObjectMeta.Name).
				WithField("currentState", checkGs.Status.State).
				WithField("awaitingState", state).Info("Waiting for states to match")

			if checkGs.Status.State == state {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			logrus.Fatalf("Wait GameServer to become Ready failed: %v", err)
		}

		err = agonesClient.AgonesV1().GameServers(gs.Namespace).Delete(ctx, newGS.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			logrus.Fatalf("Unable to delete GameServer: %v", err)
		}
	}
	return

}

// //////////////////////////////
// test
// //////////////////////////////
func createTestGameserver(c *gin.Context) {
	// config, err := getK8sConfig()
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	//kubernetesClient, err := kubernetes.NewForConfig(config)

	logger := runtime.NewLoggerWithSource("main")
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create a GameServer
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "helm-test-server-",
			// Namespace:    viper.GetString(gameserversNamespace),
			Namespace: "default",
		},
		Spec: agonesv1.GameServerSpec{
			Container: "simple-game-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 80,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "simple-game-server",
							// Image: viper.GetString(gameServerImage),
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	fmt.Println(gs)
	newGS, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatal("Unable to create GameServer: %v", err)
	}
	logrus.Infof("New GameServer name is: %s", newGS.ObjectMeta.Name)

	if viper.GetBool(isHelmTest) {
		err = wait.PollImmediate(1*time.Second, 60*time.Second, func() (bool, error) {
			checkGs, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Get(ctx, newGS.Name, metav1.GetOptions{})

			if err != nil {
				logrus.WithError(err).Warn("error retrieving gameserver")
				return false, nil
			}

			state := agonesv1.GameServerStateReady
			logger.WithField("gs", checkGs.ObjectMeta.Name).
				WithField("currentState", checkGs.Status.State).
				WithField("awaitingState", state).Info("Waiting for states to match")

			if checkGs.Status.State == state {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			logrus.Fatalf("Wait GameServer to become Ready failed: %v", err)
		}

		err = agonesClient.AgonesV1().GameServers(gs.Namespace).Delete(ctx, newGS.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			logrus.Fatalf("Unable to delete GameServer: %v", err)
		}
	}
	return
}
func createTestPVC(c *gin.Context) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	kubernetesClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	namespace := "your-namespace" // PVCを作成するNamespaceを指定してください
	pvcName := "test-go-client"
	storageClassName := "longhorn"
	storageSize := "10Gi"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}
	createdPVC, err := kubernetesClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created PVC: %s\n", createdPVC.Name)

	return
}
