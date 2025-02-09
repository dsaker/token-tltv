# token-tltv

TalkLikeTv is a language learning application designed to address limitations I’ve encountered 
in other popular language learning apps, such as Pimsleur, Babbel, and Duolingo. While these 
tools serve as strong foundational resources, I’ve found that they tend to plateau once reaching 
an intermediate level. Currently, I can understand French and Spanish well enough to follow 
audiobooks and read at a high level, but I still face challenges in expressing myself and 
comprehending native speakers during travel.

To overcome these barriers, I’ve created an application that generates a Pimsleur-like audio 
course from any file the user selects. Personally, I use subtitles from current TV shows from 
the countries I plan to visit. This approach has several benefits: it familiarizes me with 
contemporary slang, improves my understanding of spoken dialogue, and challenges me to 
express myself more naturally. Practicing with these audio files not only enhances comprehension 
of the shows but also provides an immersive, effective way to advance my language skills.

token-tltv is a simplified version of [echo-oapi-tltv](https://github.com/dsaker/echo-oapi-tltv) 
can be deployed to GCP Cloud Run. This approach reduces operational costs and provides the flexibility to share access by 
distributing tokens to selected users, removing the need for a formal authentication process.

### Required Tools

- Install [Docker](https://docs.docker.com/engine/install/)
- Install [GoLang](https://go.dev/doc/install)
- Create [Google Cloud Account](https://console.cloud.google.com/getting-started?pli=1)
- Install [gcloud CLI](https://cloud.google.com/sdk/docs/install)
- Setup [GCP ADC](https://cloud.google.com/docs/authentication/external/set-up-adc)
- Create a [Google Cloud Project](https://developers.google.com/workspace/guides/create-project)
- Install [ffmpeg](https://www.ffmpeg.org/download.html)
- Install [make] 
- Run below commands to sign in and enable the necessary Google Cloud API's
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
### Add Tokens to Firestore.<br>
when the plaintext tokens are output to the terminal copy them...<br>
these are what you will use to create mp3 files and will not be available after this step
```
cd ../../..
make coins number=10
```

### Run Container Locally
- Set up [ADC](https://cloud.google.com/docs/authentication/set-up-adc-containerized-environment)
- Install [minikube](https://minikube.sigs.k8s.io/docs/handbook/addons/gcp-auth/)
```
minikube start
minikube addons enable gcp-auth
make build/local
kubectl apply -f docker/deployment.yaml
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
