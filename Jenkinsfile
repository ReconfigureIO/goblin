node ('docker'){
  stage 'Build and Test'
  checkout scm
  sh 'GOPATH=$(pwd) go test'
 }
