let map, markers = {};

function initMap() {
    map = L.map('map').setView([37.6, -77.5], 11);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png').addTo(map);
    loadHenricoOutline();
}

function loadHenricoOutline() {
    fetch('/henrico.geojson')
        .then(response => response.json())
        .then(data => {
            L.geoJSON(data, {
                style: {
                    color: "#ff0000",
                    weight: 2,
                    opacity: .5
                }
            }).addTo(map);
        })
        .catch(error => console.error('Error loading GeoJSON:', error));
}

function addIncident(incident) {
    let marker = L.marker([incident.Location.Lat, incident.Location.Lng]).addTo(map);
    marker.bindPopup(`<b>${incident.Type}</b><br>${incident.Block}<br>${incident.Received}`);
    markers[incident.ID] = marker;

    marker.on('click', () => highlightSidebarIncident(incident.ID));

    let listItem = $(`<div class="item" data-id="${incident.ID}">
        <i class="map marker icon"></i>
        <div class="content">
            <div class="header">${incident.Type}</div>
            <div class="description">
                Block: ${incident.Block}<br>
                Received: ${incident.Received}<br>
                Status: ${incident.CallStatus}<br>
                District: ${incident.Distr}
            </div>
        </div>
    </div>`);

    $('#incident-list').append(listItem);
    listItem.on('click', function() {
        let incidentId = $(this).data('id');
        highlightMarker(incidentId);
        highlightSidebarIncident(incidentId);
    });
}

function highlightMarker(incidentId) {
    let marker = markers[incidentId];
    if (marker) {
        marker.openPopup();
        map.panTo(marker.getLatLng(), 14);
    }
}

function highlightSidebarIncident(incidentId) {
    $('#incident-list .item').removeClass('highlighted');
    let selectedItem = $(`#incident-list .item[data-id="${incidentId}"]`);
    selectedItem.addClass('highlighted');
    selectedItem.get(0).scrollIntoView({behavior: 'smooth', block: 'nearest', inline: 'start'});
}

function fetchIncidents() {
    $.getJSON('https://TPD/getAlerts')
        .then(data => Object.values(data).forEach(addIncident))
        .catch(error => console.error('Error fetching incidents:', error));
}

$(document).ready(() => {
    initMap();
    fetchIncidents();
});