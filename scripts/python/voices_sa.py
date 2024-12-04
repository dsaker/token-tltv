import json
import requests
from google.oauth2 import service_account
from google.auth.transport.requests import Request

"""voices_sa.py

voices_sa.py makes a request to google text-to-speech and downloads all of the 
supported voices in the cloud text-to-speech api. 

for more info -> https://cloud.google.com/text-to-speech/docs/voices
and https://cloud.google.com/text-to-speech/docs/reference/rest/v1/voices/list

you must create a service account to perform this request ->
https://cloud.google.com/iam/docs/service-accounts-create

db schema is at db/migrations/voices
"""

# Path to your service account key file
SERVICE_ACCOUNT_FILE = '/Users/dustysaker/Downloads/cloud-translate-439716-f4f5b53be140.json'

# Define the required scope for Text-to-Speech
SCOPES = ['https://www.googleapis.com/auth/cloud-platform']


def main():
    # Create a credentials object from the service account file
    credentials = service_account.Credentials.from_service_account_file(
        SERVICE_ACCOUNT_FILE, scopes=SCOPES)

    # Refresh the credentials to get an access token
    auth_req = Request()
    credentials.refresh(auth_req)
    access_token = credentials.token

    # Set up the headers with the access token for authorization
    headers = {
        'Authorization': f'Bearer {access_token}',
        'Content-Type': 'application/json'
    }

    # Define the API endpoint to get the available voices
    url = 'https://texttospeech.googleapis.com/v1/voices'

    # Make the API request
    response = requests.get(url, headers=headers)

    # Check and print the response
    if response.status_code == 200:
        voices = response.json()
        print(json.dumps(voices, indent=2))
    else:
        print(f"Failed to retrieve voices. Status code: {response.status_code}")
        print(response.text)


if __name__ == "__main__":
    main()
