package deployment

import (
	"github.com/Qihoo360/wayne/src/backend/client"
	"github.com/Qihoo360/wayne/src/backend/resources/common"
	"github.com/Qihoo360/wayne/src/backend/resources/event"
	"github.com/Qihoo360/wayne/src/backend/resources/pod"
	"github.com/Qihoo360/wayne/src/backend/util/maps"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Deployment struct {
	ObjectMeta common.ObjectMeta `json:"objectMeta"`
	Pods       common.PodInfo    `json:"pods"`
}

func GetDeploymentResource(cli *kubernetes.Clientset, deployment *v1beta1.Deployment) (*common.ResourceList, error) {
	old, err := cli.AppsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metaV1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return common.DeploymentResourceList(deployment), nil
		}
		return nil, err
	}
	oldResourceList := common.DeploymentResourceList(old)
	newResourceList := common.DeploymentResourceList(deployment)

	return &common.ResourceList{
		Cpu:    newResourceList.Cpu - oldResourceList.Cpu,
		Memory: newResourceList.Memory - oldResourceList.Memory,
	}, nil
}

func CreateOrUpdateDeployment(cli *kubernetes.Clientset, deployment *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	old, err := cli.AppsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name, metaV1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return cli.AppsV1beta1().Deployments(deployment.Namespace).Create(deployment)
		}
		return nil, err
	}
	old.Labels = maps.MergeLabels(old.Labels, deployment.Labels)
	oldTemplateLabels := old.Spec.Template.Labels
	old.Spec = deployment.Spec
	old.Spec.Template.Labels = maps.MergeLabels(oldTemplateLabels, deployment.Spec.Template.Labels)
	return cli.AppsV1beta1().Deployments(deployment.Namespace).Update(old)
}
func UpdateDeployment(cli *kubernetes.Clientset, deployment *v1beta1.Deployment) (*v1beta1.Deployment, error) {
	return cli.AppsV1beta1().Deployments(deployment.Namespace).Update(deployment)
}

func GetDeployment(cli *kubernetes.Clientset, name, namespace string) (*v1beta1.Deployment, error) {
	return cli.AppsV1beta1().Deployments(namespace).Get(name, metaV1.GetOptions{})
}

func GetDeploymentDetail(cli *kubernetes.Clientset, indexer *client.CacheIndexer, name, namespace string) (*Deployment, error) {
	deployment, err := cli.ExtensionsV1beta1().Deployments(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	result := &Deployment{
		ObjectMeta: common.NewObjectMeta(deployment.ObjectMeta),
	}

	podInfo := common.PodInfo{}
	podInfo.Current = deployment.Status.AvailableReplicas
	podInfo.Desired = *deployment.Spec.Replicas

	podSelector := labels.SelectorFromSet(deployment.Spec.Template.Labels).String()
	pods, err := pod.GetPodsBySelector(cli, namespace, podSelector)
	if err != nil {
		return nil, err
	}

	podInfo.Warnings = event.GetPodsWarningEvents(indexer, pods)

	result.Pods = podInfo

	return result, nil
}

func DeleteDeployment(cli *kubernetes.Clientset, name, namespace string) error {
	deletionPropagation := metaV1.DeletePropagationBackground
	return cli.ExtensionsV1beta1().
		Deployments(namespace).
		Delete(name, &metaV1.DeleteOptions{PropagationPolicy: &deletionPropagation})
}
