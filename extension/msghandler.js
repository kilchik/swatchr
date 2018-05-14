
// Listen to messages from the payload.js script and write to popout.html
chrome.runtime.onMessage.addListener(function (request) {
    console.log("onmessage");

    let title = request.title;
    // console.log(request.content);
    let foo = request.content;
    console.log("foo", foo);
    let found = foo.match(/(magnet:.+)" class.*title="/);
    console.log("found", found);
    if (found == null) {
        console.log("no magnet link found");
        return;
    }
    console.log(found);
    let magnet = request.content.match(/(magnet:.+)" class.*title="/)[1];

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
        console.log("after open");
        req.onreadystatechange = function() {   // Define event listener
            console.log("onreadystatechange");
            // If the request is compete and was successful
            if (req.readyState === 4) {
                switch (req.status) {
                    case 201:
                        setPopupMessage("Added successfully!");
                        let mainPage = `http://${addr}`;
                        chrome.tabs.create({url: mainPage});
                        return;
                    case 409:
                        setPopupMessage("Already added!");
                        return;
                }

            }
        };
        console.log("sending");
        req.send(`{"title":"${title}", "magnet":"${magnet}"}`);
    });
});
