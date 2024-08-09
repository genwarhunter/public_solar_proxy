document.addEventListener('DOMContentLoaded', function() {
    fetch('/api/v2/getConfig')
        .then(response => response.json())
        .then(data => {
            for (const key in data) {
                const element = document.getElementsByName(key)[0];
                if (element) {
                    if (element.type === 'checkbox') {
                        element.checked = data[key];
                    } else {
                        element.value = data[key];
                    }
                }
            }
        });
});

function submitSettings() {
    const form = document.getElementById('settingsForm');
    const formData = new FormData(form);
    const object = {};
    formData.forEach((value, key) => {
        if (document.getElementsByName(key)[0].type === 'checkbox') {
            object[key] = document.getElementsByName(key)[0].checked;
        } else if (document.getElementsByName(key)[0].type === 'number') {
            object[key] = Number(value);
        } else {
            object[key] = value;
        }
    });

    fetch('/api/v2/setConfig', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(object)
    })
        .then(response => response.ok ? alert('Settings saved') : alert('Failed to save settings'))
        .catch(error => alert('Error: ' + error));
}