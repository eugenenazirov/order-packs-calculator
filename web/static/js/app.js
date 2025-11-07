const api = {
  health: '/api/health',
  packSizes: '/api/pack-sizes',
  calculate: '/api/calculate',
};

const defaultPackSizes = [250, 500, 1000, 2000, 5000];

document.addEventListener('DOMContentLoaded', () => {
  const packSizesForm = document.querySelector('#pack-sizes-form');
  const packSizesInput = document.querySelector('#pack-sizes-input');
  const packSizesStatus = document.querySelector('#pack-sizes-status');
  const packSizesList = document.querySelector('#pack-sizes-list');
  const packSizesUpdatedAt = document.querySelector('#pack-sizes-updated-at');
  const packSizesReset = document.querySelector('#reset-pack-sizes');

  const calculateForm = document.querySelector('#calculate-form');
  const itemsInput = document.querySelector('#items-input');
  const calculateStatus = document.querySelector('#calculate-status');
  const resultsBody = document.querySelector('#results-body');
  const resultRowTemplate = document.querySelector('#result-row-template');
  const totalPacksEl = document.querySelector('#total-packs');
  const totalItemsEl = document.querySelector('#total-items');
  const remainderEl = document.querySelector('#remainder');
  const calculationTimeEl = document.querySelector('#calculation-time');
  const currentYear = document.querySelector('#current-year');

  currentYear.textContent = new Date().getFullYear().toString();

  loadPackSizes();

  packSizesForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    clearStatus(packSizesStatus);

    const parsed = parsePackSizes(packSizesInput.value);
    if (parsed.error) {
      showStatus(packSizesStatus, parsed.error, 'error');
      return;
    }

    try {
      showStatus(packSizesStatus, 'Updating pack sizes…', 'info');
      const response = await fetch(api.packSizes, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ packSizes: parsed.values }),
      });

      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.details || payload.error || 'Unable to update pack sizes');
      }

      showStatus(packSizesStatus, payload.message || 'Pack sizes updated successfully.', 'success');
      packSizesInput.value = payload.packSizes.join(', ');
      applyPackSizes(payload);
    } catch (error) {
      showStatus(packSizesStatus, error.message || 'Failed to update pack sizes.', 'error');
    }
  });

  packSizesReset.addEventListener('click', async () => {
    packSizesInput.value = defaultPackSizes.join(', ');
    packSizesForm.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }));
  });

  calculateForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    clearStatus(calculateStatus);

    const items = Number(itemsInput.value);
    if (!Number.isInteger(items) || items < 0) {
      showStatus(calculateStatus, 'Please enter a non-negative integer.', 'error');
      return;
    }

    try {
      showStatus(calculateStatus, 'Calculating…', 'info');
      const response = await fetch(api.calculate, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items }),
      });
      const payload = await response.json();

      if (!response.ok) {
        throw new Error(payload.details || payload.error || 'Unable to calculate packs');
      }

      showStatus(calculateStatus, 'Calculation completed.', 'success');
      renderResults(payload);
    } catch (error) {
      showStatus(calculateStatus, error.message || 'Calculation failed.', 'error');
      renderResults();
    }
  });

  async function loadPackSizes() {
    try {
      const response = await fetch(api.packSizes);
      const payload = await response.json();

      if (!response.ok) {
        throw new Error(payload.details || payload.error || 'Unable to load pack sizes');
      }

      packSizesInput.value = payload.packSizes.join(', ');
      applyPackSizes(payload);
    } catch (error) {
      showStatus(packSizesStatus, error.message || 'Failed to load pack sizes.', 'error');
    }
  }

  function applyPackSizes(payload) {
    renderPillList(packSizesList, payload.packSizes);
    packSizesUpdatedAt.textContent = payload.updatedAt
      ? `Last updated: ${formatTimestamp(payload.updatedAt)}`
      : '';
  }

  function renderPillList(container, values) {
    container.innerHTML = '';
    if (!values || values.length === 0) {
      const empty = document.createElement('li');
      empty.textContent = 'No pack sizes available';
      container.appendChild(empty);
      return;
    }

    values.forEach((value) => {
      const pill = document.createElement('li');
      pill.textContent = value.toString();
      container.appendChild(pill);
    });
  }

  function renderResults(result) {
    resultsBody.innerHTML = '';

    if (!result || !result.packs || Object.keys(result.packs).length === 0) {
      const row = document.createElement('tr');
      row.classList.add('placeholder');
      const cell = document.createElement('td');
      cell.colSpan = 2;
      cell.textContent = 'No results to display.';
      row.appendChild(cell);
      resultsBody.appendChild(row);
      totalPacksEl.textContent = '';
      totalItemsEl.textContent = '';
      remainderEl.textContent = '';
      calculationTimeEl.textContent = '';
      return;
    }

    Object.entries(result.packs)
      .sort((a, b) => Number(b[0]) - Number(a[0]))
      .forEach(([pack, quantity]) => {
        const clone = resultRowTemplate.content.cloneNode(true);
        clone.querySelector('.pack-size').textContent = `${pack}`;
        clone.querySelector('.pack-quantity').textContent = `${quantity}`;
        resultsBody.appendChild(clone);
      });

    totalPacksEl.textContent = `Total packs: ${result.totalPacks}`;
    totalItemsEl.textContent = `Total items: ${result.totalItems}`;
    remainderEl.textContent = `Remainder: ${result.remainder}`;
    calculationTimeEl.textContent = `Calculation time: ${result.calculationTimeMs} ms`;
  }
});

function parsePackSizes(value) {
  if (!value || typeof value !== 'string') {
    return { error: 'Please enter at least one pack size.' };
  }

  const parts = value.split(',').map((part) => part.trim()).filter(Boolean);
  if (parts.length === 0) {
    return { error: 'Please enter at least one pack size.' };
  }

  const sizes = [];
  const seen = new Set();

  for (const part of parts) {
    const size = Number(part);
    if (!Number.isInteger(size) || size <= 0) {
      return { error: 'Pack sizes must be positive integers.' };
    }
    if (!seen.has(size)) {
      seen.add(size);
      sizes.push(size);
    }
  }

  if (sizes.length > 10) {
    return { error: 'You can specify at most 10 distinct pack sizes.' };
  }

  return { values: sizes };
}

function formatTimestamp(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function showStatus(element, message, variant = 'info') {
  if (!element) return;
  element.textContent = message;
  element.classList.remove('success', 'error', 'info');
  if (variant !== 'info') {
    element.classList.add(variant);
  }
}

function clearStatus(element) {
  if (!element) return;
  element.textContent = '';
  element.classList.remove('success', 'error', 'info');
}
