# pulsar-configuration
How I configure pulsar on on-premise k8s with helm.
- How to install [helm chart, fluxCD]
- How to use [CLI tools, Go language]
- Tips 
  - Expose with service, not ingress 
  - How to fix metallb assining ip on specific service 

## Environment

#### Installation
```
kubernetes : v1.25.4
helm : v3.10.3
pulsar : v2.10.2

NFS Volume Provisioner : OpenEBS
Network load balancers : MetalLB
```

#### configuration 
```
cert : false
auth(jwt) : true 
pulsar_manager : false 

http: 1080   [port for admin]
pulsar: 6650 [port for consume, produce]

k8s namespace : pulsar
helm releaseName : pulsar
```

#### Test tenants, namespaces, topic
```
tenant : apache
namespace : apache/pulsar
topic : apache/pulsar/test-topic   [4 partitions]
```

## How to install

### Way 1) Using Helm

#### 1. Add helm repo (https://github.com/apache/pulsar-helm-chart)
```
helm repo add apache https://pulsar.apache.org/charts
helm repo update
```

#### 2. Create namespace ‘pulsar’ or you can use helm value.   
From now, I'll explain it according to my configuration as ‘pulsar’.

```
$kubectl create namespace pulsar

or customize with helm repo values bellow

namespace: pulsar
namespaceCreate: false
```

#### 3. Execute prepare_helm_release

```
git clone https://github.com/apache/pulsar-helm-chart
cd pulsar-helm-chart
```

execute `prepare_helm_release.sh` with your namespace and release name.

```
./scripts/pulsar/prepare_helm_release.sh -n pulsar -k pulsar --symmetric

# ./scripts/pulsar/prepare_helm_release.sh -n ${namespace} -k ${release_name} --symmetri
```

#### 4. Install helm with values file

values file : src/helm-values/pulsar-values.yaml 

```
cd src/helm-values
helm install pulsar apache/pulsar --timeout 10m -f pulsar-values.yaml
```


### Way 2) Using FluxCD

values file : src/fluxcd-helm/pulsar.yaml

1. FluxCD : HelmRepository, HelmRelease
2. Execute prepare_helm_release

```
git clone https://github.com/apache/pulsar-helm-chart
cd pulsar-helm-chart
./scripts/pulsar/prepare_helm_release.sh -n pulsar -k pulsar --symmetric
```


## How to use
These are how to simply use in cli tool, pulsar-toolset pod  and  with Go language

### Get access TOKEN
First of all, you can see client token key in pulsar-toolset pod
```
cat /pulsar/tokens/client/token
```

### CLI tool
If you are using cli tool without ‘toolset pod’, you should change configure file in ('apache-pulsar-2.11.0')[https://pulsar.apache.org/download/]

```
[conf/client.conf]

# For TLS
# webServiceUrl=https://$IP:$PORT
webServiceUrl=http://$IP:$PORT

# For TLS:
# brokerServiceUrl=pulsar+ssl://$IP:$PORT
brokerServiceUrl=pulsar://$IP:$PORT

authPlugin=org.apache.pulsar.client.impl.auth.AuthenticationToken
authParams=token:$TOKEN_KEY

tlsAllowInsecureConnection=false
tlsEnableHostnameVerification=false
```

with my helm values, I’m using only JWT token without ssl, and http port for proxy is 1080, port for broker is 6650.     
You can check ip in pulsar-proxy service external ip which type is LoadBallancer.

```
[conf/client.conf]

webServiceUrl=http://${SERVICE_LB_IP}:1080
brokerServiceUrl=pulsar://${SERVICE_LB_IP}:6650

authPlugin=org.apache.pulsar.client.impl.auth.AuthenticationToken
authParams=token:$TOKEN_KEY

tlsAllowInsecureConnection=false
tlsEnableHostnameVerification=false
```

These settings are already done in pulsar-toolset pod

### Admin
create tenent
```
bin/pulsar-admin tenants create apache
````

list tenents
```
bin/pulsar-admin tenants list
```

create namespace
````
bin/pulsar-admin namespaces create apache/pulsar
````

create topic. 
In the toolset container, create a topic test-topic with 4 partitions in the namespace apache/pulsar.

```
bin/pulsar-admin topics create-partitioned-topic apache/pulsar/test-topic -p 4
```

list all the partitioned topics in the namespace apache/pulsar
```
bin/pulsar-admin topics list-partitioned-topics apache/pulsar
```

health check
```
bin/pulsar-admin brokers healthcheck
```

### Client
produce
```
bin/pulsar-client produce apache/pulsar/test-topic  -m "---------hello apache pulsar-------" -n 10
```

consume
```
bin/pulsar-client consume -s sub apache/pulsar/test-topic  -n 0
```

### Go language
```
go get -u "github.com/apache/pulsar-client-go/pulsar"
```

[consumer_example.go]
```go
package main

import (
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
)

func main() {
	fmt.Println("consumer")

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            "pulsar://$IP:$PORT",
		Authentication: pulsar.NewAuthenticationToken("$TOKEN"),
	})
	if err != nil {
		fmt.Println(err)
		panic(fmt.Errorf("could not instantiate Pulsar client: %v", err))
	}
	defer client.Close()

	var consumer pulsar.Consumer
	var msg = make(chan pulsar.ConsumerMessage, 100)
	consumer, err = client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "apache/pulsar/test-topic",
		SubscriptionName: "sub",
		Type:             pulsar.Exclusive,
		MessageChannel:   msg,
	})

	if err != nil {
		panic(fmt.Errorf("could not subscribe from Pulsar: %v", err))
	}
	defer consumer.Close()

	for msg := range msg {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Received message msgId: %#v -- content: '%s'\n", msg.ID(), string(msg.Payload()))
		consumer.Ack(msg)
	}
	if err := consumer.Unsubscribe(); err != nil {
		log.Fatal(err)
	}
}
```

[producer_example.go]
``` go
package main

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"log"
)

func main() {
	fmt.Println("producer")

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            "pulsar://$IP:$PORT",
		Authentication: pulsar.NewAuthenticationToken("$TOKEN"),
	})

	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic: "apache/pulsar/test-topic",
	})

	if err != nil {
		log.Fatal(err)
	}

	_, err = producer.Send(context.Background(), &pulsar.ProducerMessage{
		Payload: []byte("hi"),
	})
	defer producer.Close()

	if err != nil {
		fmt.Println("Failed to publish message", err)
	}
	fmt.Println("Published message")
}
```

## TIP1 :: Expose with service, not ingress

Pulsar broker uses TCP binary protocol, not HTTP/HTTPS. Ingress exposes HTTP and HTTPS routes from outside the cluster to services within the cluster. On the other hand loadBalancer services provide a unique IP address that can be used to route traffic to the service from external clients. This is typically used for services that require public access, such as web applications or API gateways.    

Of course there also are way to expose non-HTTP services with Kubernetes Ingress by using a TCP ingress controller like '[Kong-tcp-ingress](https://docs.konghq.com/kubernetes-ingress-controller/latest/guides/using-tcpingress/) or [Nginx-tcp-ingress](https://kubernetes.github.io/ingress-nginx/user-guide/exposing-tcp-udp-services/)',
however, it's important to note that exposing non-HTTP services through an Ingress is not always recommended or appropriate. Even in kubernetes official documentation also refer this.   

```
An Ingress does not expose arbitrary ports or protocols.
Exposing services other than HTTP and HTTPS to the internet typically uses a 
service of type Service.Type=NodePort or Service.Type=LoadBalancer.
```

I had to expose pulsar to the outside, but I decided to use Helm automatically generated Loadbalancer service in terms of I don't need to route with other services which is the main role of ingress, and I don't need to use the features of L7 like DNS, SSL, etc.

## TIP2 :: How to fix metalLB assining ip on specific service
https://docs.openshift.com/container-platform/4.9/networking/metallb/metallb-configure-services.html

### Way 1. Using ip pool

create Ip pool

```
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: pulsar-ip-pool
  namespace: metallb-system
spec:
  addresses:
    - 10.1.254.103/32
```

Add created ip pool on L2Advertisement

```
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default
  namespace: metallb-system
spec:
  ipAddressPools:
    - pulsar-ip-pool
```

Note with annotation on service
```
apiVersion: v1
kind: Service
metadata:
  name: pulsar-proxy-service
  annotations:
    metallb.universe.tf/address-pool: <address_pool_name>
```


### Way 2 : Using loadBalancerIP

```
apiVersion: v1
kind: Service
metadata:
  name: service_name
spec:
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
  type: LoadBalancer
  loadBalancerIP: <ip_address>
```
