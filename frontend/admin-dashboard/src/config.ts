/**
 * Central configuration for Admin Dashboard.
 * Changing the host here will update all API endpoints.
 */
export const AppConfig = {
  // Change this to your server's IP or Domain
  host: process.env.NEXT_PUBLIC_ADMIN_HOST ?? '192.168.200.252',
  
  // Derived Admin API URL (Port 8090)
  get adminBaseUrl() {
    return `http://${this.host}:8090`;
  }
};
