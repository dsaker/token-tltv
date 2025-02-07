// Take over form submission
parseForm = document.getElementById("parse-form");
parseForm.addEventListener("submit", (event) => {
    event.preventDefault();
    const formData = new FormData(parseForm);
    sendData("/v1/parse", formData);
    parseForm.style.display = "none";
    ldsDiv.style.display = "block";
});
