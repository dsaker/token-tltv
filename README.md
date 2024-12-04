# TalkLikeTv

TalkLikeTv is a language learning application designed to address limitations I’ve encountered in other popular language learning apps, such as Pimsleur, Babbel, and Duolingo. While these tools serve as strong foundational resources, I’ve found that they tend to plateau once reaching an intermediate level. Currently, I can understand French and Spanish well enough to follow audiobooks and read at a high level, but I still face challenges in expressing myself and comprehending native speakers during travel.

To overcome these barriers, I’ve created an application that generates a Pimsleur-like audio course from any file the user selects. Personally, I use subtitles from current TV shows from the countries I plan to visit. This approach has several benefits: it familiarizes me with contemporary slang, improves my understanding of spoken dialogue, and challenges me to express myself more naturally. Practicing with these audio files not only enhances comprehension of the shows but also provides an immersive, effective way to advance my language skills.

### Installation

- [Install Docker](https://docs.docker.com/engine/install/)
- [Install GoLang](https://go.dev/doc/install)
- Create [Google Cloud Account](https://console.cloud.google.com/getting-started?pli=1)
- Install the [gcloud CLI](https://cloud.google.com/sdk/docs/install)
- Setup [GCP ADC](https://cloud.google.com/docs/authentication/external/set-up-adc )
- Create a [Google Cloud Project](https://developers.google.com/workspace/guides/create-project)
- Install [ffmpeg](https://www.ffmpeg.org/download.html)
- Install [migrate](https://github.com/golang-migrate/migrate/blob/master/cmd/migrate/README.md)
- Install [psql](https://www.timescale.com/blog/how-to-install-psql-on-mac-ubuntu-debian-windows/)
- Run below commands to sign in and enable necessary Google Cloud API's
```
gcloud init
gcloud services enable texttospeech.googleapis.com
gcloud services enable translate.googleapis.com
```
- Set up the database
```
git clone https://github.com/dsaker/echo-oapi-tltv.git 
cd echo-oapi-tltv
go mod tidy
docker pull postgres
docker run -d -P -p 127.0.0.1:5433:5432 -e POSTGRES_PASSWORD="password" --name talkliketvpg postgres
echo "export TLTV_DB_DSN=postgresql://postgres:password@localhost:5433/postgres?sslmode=disable" >> .envrc
make db/migrations/up
```
- create private ecdsa key to sign json web tokens
```
openssl ecparam -name prime256v1 -genkey -noout -out /tmp/ecprivatekey.pem
echo "export PRIVATE_KEY_PATH=/tmp/ecprivatekey.pem >> .envrc
```
- start the application
```
cd ../..
make run
```
- open http://localhost:8080/swagger/ in local browser
- click on POST /users "Try it out" and create a new user
- add "titles:w" permission to user
```
make db/psql # creates a connection to the local database
select * from users; # get id for you user
insert into users_permissions values (<id from above>,2); # this adds 'titles:w' permission needed to create audio files
```
- click POST /users/login and login
- copy response body and decode it at https://www.base64decode.org/
- click on Authorize and add decoded value
- click on POST /audio/fromfile and click on "Try it out"
- *on Linux you will have to change srt file ending to txt

### To update voices or languages when google makes changes
- [Create an api key](https://cloud.google.com/docs/authentication/api-keys) to load the voices in the database
```
cd scripts/python
pip install virtualenv
python3 -m venv <myenvname>
source env/bin/activate
pip install -r requirements.txt
export API_KEY=<your api key>
export TLTV_DB_DSN=postgresql://postgres:password@localhost:5433/postgres?sslmode=disable
python create_sql.py
```
- create new migration files
```
make db/migrations/new name=<insert new voices or languages>
```
- add the insert statements from the output above to the new db/migrations files
```
make db/migrations/up
```