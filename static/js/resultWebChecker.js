// Variables for working with Ag-Grid and data
let gridApi;
const apiUrl = '/api/v2/getResultWebChecker';

// Ag-Grid table settings
const gridOptions = {
    dataTypeDefinitions: {
        ip: {
            extendsDataType: 'number',
            baseDataType: 'number',
            valueFormatter: params => `${[(params.value >>> 24) & 0xff, (params.value >>> 16) & 0xff, (params.value >>> 8) & 0xff, params.value & 0xff].join('.')}`,
        },
    },
    columnDefs: [
        { field: "proxy", headerName: "Proxy" },
        { field: "ipOut", headerName: "IPOut", cellDataType: "ip"},
        { field: "country", headerName: "Country", filter: "agTextColumnFilter" },
        { field: "latency", headerName: "Latency (ms)", filter: "agNumberColumnFilter" },
        { field: "exist", headerName: "Exists", filter: "agNumberColumnFilter" },
    ],
    defaultColDef: {
        flex: 1,
        resizable: true,
        sortable: true,
        filter: true
    },
    animateRows: true,
    rowData: [],
    onGridReady: params => {
        gridApi = params.api;
    }
};

function updateProgressBar(checked, total) {
    const percentage = total > 0 ? (checked / total) * 100 : 0;
    const formattedPercentage = percentage.toFixed(2);
    const progressElement = document.getElementById('progress');
    progressElement.style.width = `${formattedPercentage}%`;
    progressElement.textContent = `${formattedPercentage}%`;
}

// Function to get hash from URL
function getHashFromUrl() {
    const path = window.location.pathname;
    const segments = path.split('/');
    return segments.pop() || segments.pop(); // handle potential trailing slash
}

document.addEventListener("DOMContentLoaded", () => {
    const gridDiv = document.querySelector("#result");
    gridApi = agGrid.createGrid(gridDiv, gridOptions);
    const hash = getHashFromUrl();
    const fetchUrl = `${apiUrl}?hash=${hash}`;

    // Function to update data from API
    async function fetchStatus() {
        let end = false;
        try {
            let response = await fetch(fetchUrl, { cache: 'default' })
            let data = await response.json();

            const checked = data.checked || 0;
            const total = data.total || 1; // Use 1 to avoid division by zero
            const countWork = data.countWork || 0; // Add countWork field
            const countExist = data.countExist || 0

            if (Object.keys(data).length === 0){
                clearInterval(intervalId);
            }

            updateProgressBar(checked, total);

            // Update Total and Count Work
            document.getElementById('total').textContent = total;
            document.getElementById('countWork').textContent = data.proxylist.length;
            document.getElementById('countExist').textContent = countExist

            gridApi.setGridOption("rowData", data.proxylist);
            var rowHeight = 25;
            var headerHeight = 40;
            var totalHeight = headerHeight + (data.length * rowHeight);
            gridDiv.style.height = totalHeight+'px';
            if (data.status || data.total === 0) {
                end = true;
            }
        } catch (e) {
            console.log(e)
        } finally {
            if (!end) {
                setTimeout(fetchStatus, 1000);
            }
        }


    }

    const intervalId = setTimeout(fetchStatus, 1000);

});
