package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	// "os"
	"time"
	"net/http"
	// "encoding/json"


	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	// "agones.dev/agones/pkg/util/runtime"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	// "github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/util/wait"

)

const (
	gameServerImage      = "GAMESERVER_IMAGE"
	isHelmTest           = "IS_HELM_TEST"
	mcServersNamespace 	 = "default"
	mcServerName		 = "mc-server-"
	mcPvcName			 = "mc-pvc-"
)

var kubernetesClient *kubernetes.Clientset
var agonesClient *versioned.Clientset



func main() {
	config, err := getK8sConfig()
	kubernetesClient, err = kubernetes.NewForConfig(config)
	if err != nil {

	}
	agonesClient, err = versioned.NewForConfig(config)
	if err != nil {

	}
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "healthy",
		})
	})
	r.GET("/create", createMcServer)
	r.GET("/delete",deleteMcServer)
	r.GET("/statusstream", streamGsStatus)
	r.GET("/status",gsStatus)


	// r.GET("/pvc", createPVC)
	r.Run()
}
func getPodStatus(podName, namespace string) (string, error) {
	pod, err := kubernetesClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(pod.Status.Phase), nil
}

func gsStatus( c *gin.Context){
	userid := c.Query("userid")
	currentGs, err := agonesClient.AgonesV1().GameServers("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "userid=" + userid,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		panic(err.Error())
	}
	fmt.Printf("\ncurrentGs\n:%v",currentGs.Items)
	gsStatuses := make([]map[string]interface{},0)

	for _,gs := range currentGs.Items {
		gsStatus := make(map[string]interface{})
		gsStatus["gameserver"] = gs
		pod, err := kubernetesClient.CoreV1().Pods(mcServersNamespace).Get(context.TODO(),gs.Name,metav1.GetOptions{})
		if err != nil{
			fmt.Printf("error fail get pod status:%v",err)
			gsStatus["pod"] = "Pod not found"
		}else {
			gsStatus["pod"] = pod
		}
		gsStatuses = append(gsStatuses,gsStatus)
	}

	c.JSON(http.StatusOK, gin.H{
		"gsStatus":gsStatuses,
	})
}


func streamGsStatus(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	fmt.Println("Start stream GS Status")

	userid := c.Query("userid")
	closeNotifier := c.Writer.CloseNotify()

	// lastGs := map[string]agonesv1.GameServerState{}
	fmt.Println("start for")
	for {
		select {
			case <-closeNotifier:
				fmt.Println("client has disconnected")
				return
			default:
				currentGs, err := agonesClient.AgonesV1().GameServers("").List(context.TODO(), metav1.ListOptions{
					LabelSelector: "userid=" + userid,
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					fmt.Printf("get k8s status error:%v",err)
					// panic(err.Error())
				}
				fmt.Printf("\ncurrentGs\n:%v",currentGs.Items)
				gsStatuses := make([]map[string]interface{},0)

				for _,gs := range currentGs.Items {
					gsStatus := make(map[string]interface{})
					gsStatus["gameserver"] = gs
					pod, err := kubernetesClient.CoreV1().Pods(mcServersNamespace).Get(context.TODO(),gs.Name,metav1.GetOptions{})
					if err != nil{
						fmt.Printf("error fail get pod status:%v",err)
						gsStatus["pod"] = "Pod not found"
					}else {
						gsStatus["pod"] = pod
					}
					gsStatuses = append(gsStatuses,gsStatus)
				}

		
				// for _,gs := range curren。tGs.Items {
					// lp := lastGs[gs.Name]
					// if gs.Status.State != lp {
						// jsonData, err := json.Marshal(currentGs.Items)
						if err != nil {
							// エラー処理
							fmt.Printf("JSONエンコードエラー: %v", err)
							continue
						}
						c.SSEvent("gsStatus", gsStatuses)
						fmt.Println("#########################send#######################")
						c.Writer.(http.Flusher).Flush()
					// }
					// lastGs[gs.Name] = gs.Status.State
				// }
				time.Sleep(1 * time.Second)					
		}
	}
}
func getPodLog(){

}

func teststatus(c *gin.Context){
    fmt.Printf("Pod created %v","feij")
	
	pod := &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name: "example-pod",
        },
        Spec: corev1.PodSpec{
            Containers: []corev1.Container{
                {
                    Name:  "example-container",
                    Image: "busybox",
                    Args:  []string{"sh", "-c", "echo Hello Kubernetes! && sleep 3600"},
                },
            },
        },
    }

    podsClient := kubernetesClient.CoreV1().Pods("default")
    _, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
    if err != nil {
        panic(err.Error())
    }
    fmt.Println("Pod created")
	err = wait.PollImmediate(5*time.Second, 60*time.Second, func() (bool, error) {
        pod, err := kubernetesClient.CoreV1().Pods(mcServersNamespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
        if err != nil {
            return false, err
        }

        fmt.Printf("Pod %s is in %s state\n", pod.Name, pod.Status.Phase)
        if pod.Status.Phase == "Running" {
            return true, nil
        }

        return false, nil
    })

    if err != nil {
        fmt.Printf("Error while monitoring pod status: %v\n", err)
    } else {
        fmt.Printf("Pod %s is now running.\n", pod.Name)
    }
	

    fmt.Println("Finished streaming pod logs")
}


func createPVC(userid string, storageClassName string, storageSize string) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mcPvcName,
			Labels: map[string]string{
				"userid": userid, // ラベルを追加
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}
	createdPVC, err := kubernetesClient.CoreV1().PersistentVolumeClaims(mcServersNamespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	fmt.Println("##############createdPVC###################",createdPVC)
	if err != nil {
		return nil, err
	}
	return createdPVC, nil
}
func createMcGs(userid string,pvcName string,sname string)(*agonesv1.GameServer, error){
		// Create a GameServer

		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: mcServerName,
				Labels: map[string]string{
					"userid": userid, // ラベルを追加
					"sname": sname,
				},
			},
			Spec: agonesv1.GameServerSpec{
				Container: "mc-server",
				Ports: []agonesv1.GameServerPort{
					{
						Name:          "mc",
						PortPolicy:    "Dynamic",
						ContainerPort: 25565,
						Protocol:      "TCP",
					},
				},
				Health: agonesv1.Health{
					InitialDelaySeconds: 120,
					PeriodSeconds:       12,
					FailureThreshold:    5,
				},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "mc-server",
								Image: "itzg/minecraft-server",
								Env: []corev1.EnvVar{
									{
										Name:  "EULA",
										Value: "TRUE",
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/data",
										Name:      "world-vol",
									},
								},
								Ports: []corev1.ContainerPort{
									{ContainerPort: 25575},
								},
							},
							{
								Name:  "mc-monitor",
								Image: "saulmaldonado/agones-mc",
								Args:            []string{"monitor"},
								Env: []corev1.EnvVar{
									{
										Name:  "INITIAL_DELAY",
										Value: "60s",
									},
									{
										Name:  "MAX_ATTEMPTS",
										Value: "5",
									},
									{
										Name:  "INTERVAL",
										Value: "10s",
									},
									{
										Name:  "TIMEOUT",
										Value: "10s",
									},
								},
								ImagePullPolicy: "Always",
							},
							{
								Name:  "mc-backup",
								Image: "saulmaldonado/agones-mc",
								Args:  []string{"backup"},
								Env: []corev1.EnvVar{
									{
										Name:  "BUCKET_NAME",
										Value: "agones-minecraft-mc-worlds",
									},
									{
										Name:  "BACKUP_CRON",
										Value: "0 */6 * * *",
									},
									{
										Name:  "INITIAL_DELAY",
										Value: "60s",
									},
									{
										Name: "POD_NAME",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "metadata.name",
											},
										},
									},
									{
										Name:  "RCON_PASSWORD",
										Value: "minecraft",
									},
								},
								ImagePullPolicy: "Always",
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/data",
										Name:      "world-vol",
									},
								},
							},
							// {
							// 	Name:  "mc-fileserver",
							// 	Image: "saulmaldonado/agones-mc",
							// 	Args:  []string{"fileserver"},
							// 	Env: []corev1.EnvVar{
							// 		{
							// 			Name:  "VOLUME",
							// 			Value: "/data",
							// 		},
							// 	},
							// 	ImagePullPolicy: "Always",
							// 	VolumeMounts: []corev1.VolumeMount{
							// 		{
							// 			MountPath: "/data",
							// 			Name:      "world-vol",
							// 		},
							// 	},
							// },
						},
						Volumes: []corev1.Volume{
							{
								Name: "world-vol",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvcName,
									},
								},
							},
						},
					},
				},
			},
		}
		ctx := context.Background()
		newGS, err := agonesClient.AgonesV1().GameServers("default").Create(ctx, gs, metav1.CreateOptions{})
		if err != nil {
			logrus.Fatal("Unable to create GameServer: %v", err)
		}
		return newGS, err
}

func getK8sConfig() (*rest.Config, error) {
	// InClusterConfigの取得処理
	config, err := rest.InClusterConfig()
	// k8s Cluster内でないなら、エラーが返ってくる
	// その場合は次のローカルのkubeconfig取得処理へ
	if err == nil {
		return config, nil
	}
	// ローカルで開発するため用
	return clientcmd.BuildConfigFromFlags("https://10.0.1.41:6443", "/go/src/config")
}

func createMcServer(c *gin.Context) {
	// useridを受け取る
	userid := c.Query("userid")
	sname := c.Query("sname")
	fmt.Print(sname)

	createdPVC, err := createPVC(userid, "longhorn", "3Gi")
	if err != nil {
		fmt.Printf("can't create pvc:%v",err)
	}
	createdMcGs, err := createMcGs(userid,createdPVC.Name,sname)
	if err != nil {
		fmt.Printf("can't create gameserver:%v",err)
	}

	fmt.Println(createMcGs)
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"createdPVC":createdPVC,
		"createdGS":createdMcGs,
	})
	return 
}

func deleteMcServer(c *gin.Context){
	userid := "1"
	podName := c.Query("podname")
	if podName != ""{
		deleteMcGs(podName,userid)
	}
	pvcName:= c.Query("pvcname")
	if pvcName != ""{
		deletePVC(pvcName,userid)
	}


	// err := agonesClient.AgonesV1().GameServers(mcServersNamespace).Delete(ctx, podName, metav1.DeleteOptions{})
	// if err != nil {
	// 	logrus.Fatalf("Unable to delete GameServer: %v", err)
	// }
	// pvcName:= c.Query("pvcname")
	// err := deletePVC(kubernetesClient, mcServersNamespace, PvcName, userid, "longhorn", "10Gi")
	// if podname != '' {
	// 	deletePVC(kubernetesClient,mcServersNamespace,)
	// }
}
func deletePVC( name string, id string) {
	deletePolicy := metav1.DeletePropagationForeground // Define deletePolicy
	err := kubernetesClient.CoreV1().PersistentVolumeClaims(mcServersNamespace).Delete(context.TODO(), name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
	//  fmt.Fprintf(os.Stderr, "Error deleting pvc: %v\n", err)
	fmt.Printf("error delete pvc:%v",err)

	}
   return 
}
func deleteMcGs(name string,id string) {
	ctx := context.Background()

	err := agonesClient.AgonesV1().GameServers(mcServersNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		// logrus.Fatalf("Unable to delete GameServer: %v", err)
		fmt.Printf("error delete GS:%v",err)
	}
}
