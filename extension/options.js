function save_options() {
    let addr = document.getElementById("srv-addr").value;

    let req = new XMLHttpRequest();
    req.timeout = 2000;
    req.open('GET', `http://${addr}/ping`, true);

    let onerr = function () {
        let statusMsg = document.getElementById('status');
        statusMsg.innerHTML = `<span style="color:red">Failed to connect!</span>`;
    };
    req.onload = function () {
        if (req.status === 200) {
            chrome.storage.sync.set({
                addr: addr,
            }, function() {
                let statusMsg = document.getElementById('status');
                statusMsg.innerHTML = `<span style="color:green">OK!</span>`;
                setTimeout(function() {
                    statusMsg.innerHTML = '';
                }, 2000);
            });
        } else {
            onerr();
        }
    };
    req.onerror = onerr();
    req.send();
}

function restore_options() {
    chrome.storage.sync.get({
        addr: '',
    }, function(items) {
        document.getElementById('srv-addr').value = items.addr;
    });
}
document.addEventListener('DOMContentLoaded', restore_options);
document.getElementById('try-button').addEventListener('click', save_options);
