module kubecube-webconsole

go 1.15

require (
	github.com/astaxie/beego v1.12.3
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emicklei/go-restful v2.15.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kubecube-io/kubecube v0.0.0-20210511114933-04c602bdca67
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/shiena/ansicolor v0.0.0-20200904210342-c7312218db18 // indirect
	gopkg.in/igm/sockjs-go.v2 v2.1.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	sigs.k8s.io/controller-runtime v0.8.3
)
