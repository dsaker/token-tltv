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
designed for deployment on GCP Cloud Run. It eliminates the need for a database and uses token-
based access instead of traditional authentication. This approach reduces operational costs and 
provides the flexibility to share access by distributing tokens to selected users, removing the 
need for a formal authentication process.

### Installation Locally

- [Install Docker](https://docs.docker.com/engine/install/)
- [Install GoLang](https://go.dev/doc/install)
- Create [Google Cloud Account](https://console.cloud.google.com/getting-started?pli=1)
- Install the [gcloud CLI](https://cloud.google.com/sdk/docs/install)
- Setup [GCP ADC](https://cloud.google.com/docs/authentication/external/set-up-adc )
- Create a [Google Cloud Project](https://developers.google.com/workspace/guides/create-project)
- Install [ffmpeg](https://www.ffmpeg.org/download.html)
- Run below commands to sign in and enable necessary Google Cloud API's
```
gcloud init
gcloud services enable artifactregistry.googleapis.com
gcloud services enable texttospeech.googleapis.com
gcloud services enable translate.googleapis.com
```
- start the application
```
cd ../..
make run
```

### To update voices or languages when google makes changes
- [Create an api key](https://cloud.google.com/docs/authentication/api-keys) to load the voices in the database
// todo
