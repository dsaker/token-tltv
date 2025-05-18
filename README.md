# token-tltv

TalkLikeTV is a language learning application I developed to address the limitations I’ve experienced with 
popular apps like Pimsleur, Babbel, and Duolingo. While these tools offer a solid foundation, I found they 
often fall short at the intermediate level, where progress tends to plateau.

To overcome these challenges, I created an app that generates Pimsleur-style audio courses from any file 
the user provides—particularly useful when using subtitles from current TV shows in the target language. 
This method offers multiple advantages: it exposes users to modern slang, improves comprehension of natural 
spoken dialogue, and encourages more authentic self-expression. Practicing with these audio files not only 
boosts understanding of the source material but also provides a more immersive and practical way to advance language skills.

token-tltv is a simplified version of [echo-oapi-tltv](https://github.com/dsaker/echo-oapi-tltv)  It uses token-based access to stay publicly accessible without risking unexpected usage costs.

### Required Tools

- Install [Docker](https://docs.docker.com/engine/install/)
- Install [GoLang](https://go.dev/doc/install)
- Create [Google Cloud Account](https://console.cloud.google.com/getting-started?pli=1)
- Install [gcloud CLI](https://cloud.google.com/sdk/docs/install)
- Setup [GCP ADC](https://cloud.google.com/docs/authentication/external/set-up-adc)
- Create a [Google Cloud Project](https://developers.google.com/workspace/guides/create-project)
- Install [ffmpeg](https://www.ffmpeg.org/download.html)
- Install [make] 
- Run below commands to enable the necessary Google Cloud API's
```
gcloud services enable texttospeech.googleapis.com
gcloud services enable translate.googleapis.com
```

### Run locally
```
git clone git@github.com:dsaker/token-tltv.git
cd token-tltv
make run
```

### Deploy to Google Cloud Run
- Install [terraform](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli)
```
glcoud init
gcloud services enable texttospeech.googleapis.com
gcloud services enable translate.googleapis.com
gcloud services enable firestore.googleapis.com 
gcloud services enable artifactregistry.googleapis.com
gcloud services enable compute.googleapis.com
gcloud services enable run.googleapis.com
git clone git@github.com:dsaker/token-tltv.git
cd token-tltv/cloud-run/dont_erase/terraform
cp terraform.tfvars.tmpl terraform.tfvars
```
change project_id in terraform.tfvars to the project you just created<br>
you can uncomment static_ip_adress.tf if you need a static ip

```
terraform init
terraform plan
terrafrom apply -target=google_artifact_registry_repository.token_tltv
```
setup up docker auth - https://cloud.google.com/artifact-registry/docs/docker/authentication
```
gcloud auth configure-docker us-east4-docker.pkg.dev
```
build and push the docker container to the artifact registry
```
cd ../..
cp .envrc.tmpl .envrc
```
change PROJECT_ID in .envrc to the project you just created<br>
build docker image and push to artifactory registry in google cloud
```
make build/cloud
```
run docker container in cloud run<br>
cloud run url will be printed in the output
```
cd terraform/cloud-run/dont_destroy
```
if you want alerts set then add an email and phone number<br>
if you don't want alerts set, then comment out or delete alerts.tf
```
terraform apply
cd ../cloud-run/
terraform init
cp terraform.tfvars.tmpl terraform.tfvars # fill in PROJECT_ID
terraform apply
```

### Add Tokens to Firestore
when the plaintext tokens are output to the terminal copy them...<br>
these are what you will use to create mp3 files and will not be available after this step
```
cd ../../..
make upload-coins number=10
```

### Run Container Locally
- Set up [ADC](https://cloud.google.com/docs/authentication/set-up-adc-containerized-environment)
- Install [minikube](https://minikube.sigs.k8s.io/docs/handbook/addons/gcp-auth/)
```
minikube start
minikube addons enable gcp-auth
make build/local
kubectl apply -f docker/deployment.yaml
minikube image load token-tltv:latest
minikube service tltv-svc --url 
```

### Update Languages and Voices
- Google Cloud Platform
```
go run ./scripts/go/languages > internal/models/jsonmodels/google_languages.json
go run ./scripts/go/voices > internal/models/jsonmodels/google_voices.json
```
- Amazon Web Services
```
aws translate list-Languages > internal/models/jsonmodels/aws_languages.json
aws polly describe-Voices > internal/models/jsonmodels/aws_voices.json
```
