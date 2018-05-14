
function onLoaded() {
    chrome.extension.getBackgroundPage().chrome.tabs.executeScript(null, {
        file: 'payload.js'
    });
}

function setPopupMessage(text) {
    document.getElementById("msg-text").innerHTML = text;
}

document.addEventListener('DOMContentLoaded', function () {
    onLoaded();
});
