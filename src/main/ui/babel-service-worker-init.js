function onError(err) {
    console.error('Error registering service-worker:', err)
    document.getElementById('root').innerText = err.toString()
}


if ('serviceWorker' in navigator) {
    console.log('service-worker:', "initializing")
    navigator.serviceWorker.register('/babel-service-worker.js', { scope: '/' })
        .then(registration => {
            window.serviceWorkerRegistration = registration
            // use `await window.serviceWorkerRegistration.unregister()` to unregister the service worker
            if (registration.installing) {
                const sw = registration.installing || registration.waiting;
                sw.onstatechange = function () {
                    if (sw.state === 'installed') {
                        // SW installed.  Refresh page so SW can respond with SW-enabled page.
                        window.location.reload();
                    }
                };
            } else if (registration.active) {
                // something's not right or SW is bypassed.  previously-installed SW should have redirected this request to different page
                //onError(new Error('Service Worker is installed and not redirecting.'))
            }
        })
        .catch(onError)
} else {
    onError('Browser does not support service workers :-(')
}
