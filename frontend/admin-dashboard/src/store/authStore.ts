import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type User = { id: string; username: string; role: string };

type AuthState = {
  token: string | null;
  user: User | null;
  setAuth: (token: string, user: User) => void;
  logout: () => void;
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setAuth: (token, user) => {
        if (typeof window !== 'undefined') localStorage.setItem('admin_token', token);
        set({ token, user });
      },
      logout: () => {
        if (typeof window !== 'undefined') localStorage.removeItem('admin_token');
        set({ token: null, user: null });
      },
    }),
    { name: 'admin-auth' }
  )
);
