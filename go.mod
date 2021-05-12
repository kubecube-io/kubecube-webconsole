module kubecube-webconsole

go 1.15

require (
	github.com/FZambia/sentinel v1.1.0 // indirect
	github.com/astaxie/beego v1.12.3
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kubecube-io/kubecube v0.0.0-20210511114933-04c602bdca67
	github.com/patrickmn/go-cache v2.1.0+incompatible
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	gopkg.in/igm/sockjs-go.v2 v2.1.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	sigs.k8s.io/controller-runtime v0.8.3
)

replace k8s.io/api v0.21.0 => k8s.io/api v0.20.5
