declare module 'vue-sonner' {
  import type { Component } from 'vue';
  export const Toaster: Component;
  export function toast(message: string, options?: any): void;
  export namespace toast {
    function success(message: string, options?: any): void;
    function error(message: string, options?: any): void;
    function warning(message: string, options?: any): void;
    function info(message: string, options?: any): void;
  }
}
