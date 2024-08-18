document.getElementById('lumiForm').addEventListener('submit', function(e) {
    e.preventDefault();

    const formData = new FormData(this);
    const jsonData = {};

    formData.forEach((value, key) => {
        if (key === 'tag' || key === 'and' || key === 'ignore') {
            jsonData[key] = value.split(',').map(item => item.trim()).filter(item => item !== '');
        } else if (key === 'mediaCount') {
            jsonData[key] = parseInt(value) || 20;
        } else {
            jsonData[key] = value;
        }
    });

    fetch('/launch', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(jsonData),
    })
        .then(response => response.text())
        .then(data => {
            updateStatus(data);
            pollStatus();
        })
        .catch((error) => {
            console.error('Error:', error);
            updateStatus('Error: ' + error.message);
        });
});

function updateStatus(message) {
    document.getElementById('status').textContent = message;
}

function updateProgress(progress) {
    const progressContainer = document.getElementById('progress-container');
    const progressBar = document.querySelector('.progress-bar-fill');
    const progressText = document.getElementById('progress-text');

    progressContainer.style.display = 'block';
    const percentage = (progress.downloadedImages / progress.totalImages) * 100 || 0;
    progressBar.style.width = `${percentage}%`;
    progressText.textContent = `Downloaded: ${progress.downloadedImages}/${progress.totalImages} (Skipped: ${progress.skippedImages})`;
}

function pollStatus() {
    const interval = setInterval(() => {
        fetch('/status')
            .then(response => response.json())
            .then(data => {
                updateStatus(`Current status: ${data.status}`);

                if (data.progress) {
                    updateProgress(data.progress);
                }

                if (data.status === 'Completed') {
                    clearInterval(interval);
                }
            })
            .catch(error => {
                console.error('Error polling status:', error);
                clearInterval(interval);
            });
    }, 1000); // Poll every second for smoother updates
}