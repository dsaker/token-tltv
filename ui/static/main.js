const ldsDiv = document.getElementById("lds-div");
const divFlash = document.getElementById("div-flash")

let filename = ""
function sendData(url, formData) {
	divFlash.style.display = "none";
	divFlash.innerHTML = "";
	// Associate the FormData object with the form element
	fetch(url, {
		method: "POST",
		// Set the FormData instance as the request body
		body: formData,
	}).then(async (response) => {
		if (!response.ok) {
			throw Error(await response.text());
		}
		// We are reading the *Content-Disposition* header for getting the original filename given from the server
		const header = response.headers.get('Content-Disposition');
		const parts = header.split(';');
		filename = parts[1].split('=')[1].replaceAll("\"", "");
		return response.blob()
	}).then((blob) => {
			if (blob != null) {
				let url = window.URL.createObjectURL(blob);
				let a = document.createElement('a');
				a.href = url;
				a.download = filename;
				document.body.appendChild(a);
				a.click();
				a.remove();
			}
		})
		.catch((message) => {
			divFlash.style.display = "block";
			divFlash.innerHTML = message;
		});

}

