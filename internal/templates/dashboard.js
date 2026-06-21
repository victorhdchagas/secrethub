function dashboardApp() {
  return {
    view: 'home',
    vaults: [],
    vaultName: '',
    vaultData: {},
    showKeys: {},
    newVaultName: '',
    newKey: '',
    newValue: '',
    loading: false,
    toast: '',
    toastTimer: null,
    sessionExpiry: '',
    _navLock: false,

    init() {
      this.loadVaults()
      this.initHash()
      window.addEventListener('hashchange', () => this.onHashChange())
    },

    initHash() {
      const hash = location.hash
      if (hash.startsWith('#/vault/')) {
        const name = decodeURIComponent(hash.slice(8))
        if (name && name.indexOf('/') === -1) this.navigate(name)
      } else if (hash === '#/settings') {
        this.view = 'settings'
      }
    },

    onHashChange() {
      if (this._navLock) { this._navLock = false; return }
      const hash = location.hash
      if (hash.startsWith('#/vault/')) {
        const name = decodeURIComponent(hash.slice(8))
        if (name && name.indexOf('/') === -1 && name !== this.vaultName) this.navigate(name)
      } else if (hash === '#/settings') {
        this.view = 'settings'
      } else {
        this.view = 'home'
      }
    },

    async req(method, path, body) {
      this.loading = true
      try {
        const opts = { method, headers: {} }
        if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body) }
        const res = await fetch('/api' + path, opts)
        this.updateExpiry(res.headers)
        if (res.status === 401) { window.location.href = '/'; return null }
        if (!res.ok) { const err = await res.text(); this.showToast(err); return null }
        if (res.status === 204) return null
        return res.json()
      } catch {
        this.showToast('Erro de rede')
        return null
      } finally {
        this.loading = false
      }
    },

    updateExpiry(headers) {
      const expiry = headers.get('X-Session-Expires')
      if (expiry) {
        const d = new Date(expiry)
        this.sessionExpiry = 'Sessão expira às ' + d.toLocaleTimeString('pt-BR', {
          timeZone: 'America/Sao_Paulo', hour: '2-digit', minute: '2-digit'
        })
      }
    },

    showToast(msg) {
      this.toast = msg
      if (this.toastTimer) clearTimeout(this.toastTimer)
      this.toastTimer = setTimeout(() => { this.toast = '' }, 3000)
    },

    async loadVaults() {
      const data = await this.req('GET', '/vaults')
      if (data) this.vaults = data.vaults || []
    },

    async navigate(name) {
      this.vaultName = name
      this.vaultData = {}
      this.showKeys = {}
      this.view = 'vault'
      this._navLock = true
      location.hash = '#/vault/' + encodeURIComponent(name)
      this._navLock = false
      await this.loadVaultData(name)
    },

    async loadVaultData(name) {
      const data = await this.req('GET', '/vault/' + encodeURIComponent(name))
      if (data) this.vaultData = data
    },

    async createVault() {
      const name = this.newVaultName.trim()
      if (!name) return
      await this.req('POST', '/vault/' + encodeURIComponent(name))
      this.newVaultName = ''
      this.loadVaults()
    },

    async saveVault() {
      await this.req('POST', '/vault/' + encodeURIComponent(this.vaultName), this.vaultData)
      this.showToast('Salvo!')
    },

    async addKey() {
      const key = this.newKey.trim()
      if (!key) return
      this.vaultData[key] = this.newValue
      await this.saveVault()
      this.newKey = ''
      this.newValue = ''
    },

    async deleteKey(key) {
      if (!confirm('Excluir key "' + key + '"?')) return
      await this.req('DELETE', '/vault/' + encodeURIComponent(this.vaultName) + '/keys/' + encodeURIComponent(key))
      delete this.vaultData[key]
      delete this.showKeys[key]
      this.showToast('Key excluída')
    },

    toggleKey(key) {
      this.showKeys[key] = !this.showKeys[key]
    },

    async exportVault() {
      const res = await fetch('/api/vault/' + encodeURIComponent(this.vaultName) + '/export')
      this.updateExpiry(res.headers)
      if (res.status === 401) { window.location.href = '/'; return }
      if (!res.ok) return
      const blob = new Blob([await res.text()], { type: 'text/plain' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = this.vaultName + '.env'
      a.click()
    },

    async copyAll() {
      const res = await fetch('/api/vault/' + encodeURIComponent(this.vaultName) + '/export')
      this.updateExpiry(res.headers)
      if (res.status === 401) { window.location.href = '/'; return }
      if (!res.ok) return
      await navigator.clipboard.writeText(await res.text())
      this.showToast('Copiado!')
    },

    copyKey(key) {
      navigator.clipboard.writeText(this.vaultData[key])
      this.showToast('Copiado!')
    },

    showSettings() {
      this.view = 'settings'
      this._navLock = true
      location.hash = '#/settings'
      this._navLock = false
    }
  }
}
