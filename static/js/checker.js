document.addEventListener('DOMContentLoaded', () => {
    const checkButton = document.getElementById('checkButton');
    checkButton.addEventListener('click', async function() {
        var proxiesText = document.getElementById("proxiesTextarea").value;
        var proxiesArray = proxiesText.split(/\r?\n/);
        var jsonData = {
            "proxies": proxiesArray
        };

        if (jsonData.proxies.length === 0 || (jsonData.proxies.length === 1 && jsonData.proxies[0] === '')) {
            return;
        }
        try {
            let response = await fetch('/api/v2/instantCheckList', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(jsonData)
            });
            if (response.ok) {
                let responseData = await response.json();
                if (responseData.id) {
                    window.location.href = `/checker/result/${responseData.id}`;
                } else {
                    console.error('Ответ от сервера не содержит id.');
                }
            } else {
                console.error('Ошибка HTTP: ' + response.status);
            }
        } catch (error) {
            console.error('Ошибка при выполнении запроса:', error);
        }
    });
});
