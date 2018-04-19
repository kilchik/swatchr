
function onLoaded() {
    chrome.extension.getBackgroundPage().chrome.tabs.executeScript(null, {
        file: 'payload.js'
    });
}

// Listen to messages from the payload.js script and write to popout.html
chrome.runtime.onMessage.addListener(function (request) {
    let title = request.title;
    let magnet = request.content.match(/(magnet:.+)" class.*title="Скачать по magnet-ссылке">/)[1];

    let slashPos = title.indexOf("/");
    if (slashPos !== -1) {
        title = title.slice(0, slashPos);
    }

    let req = new XMLHttpRequest();
    chrome.storage.sync.get({
        addr: '',
    }, function(items) {
        let addr = items.addr;
        if (addr === "") {
            setPopupMessage("You should specify server address first!");
            return
        }
        req.open('POST', `http://${addr}/add`, true);
        req.onreadystatechange = function() {   // Define event listener
            // If the request is compete and was successful
            if (req.readyState === 4) {
                switch (req.status) {
                    case 201:
                        setPopupMessage("Added successfully!");
                        let mainPage = `http://${addr}`;
                        chrome.tabs.create({url: mainPage});
                        break;
                    case 409:
                        setPopupMessage("Already added!");
                        break;
                }
            }
        };
        req.send(`{"title":"${title}", "magnet":"${magnet}"}`);
    });
});

function setPopupMessage(text) {
    document.getElementById("msg-text").innerHTML = text;
}

document.addEventListener('DOMContentLoaded', function () {
    onLoaded();
});
