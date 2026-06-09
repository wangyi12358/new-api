/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import termsContent from './kovarTermsOfUse.md?raw';

export const KOVAR_TERMS_VERSION = '2026-03-25';
export const KOVAR_TERMS_STORAGE_KEY = 'kovar_terms_accepted_version';

export { termsContent };

export function hasAcceptedKovarTerms() {
  if (typeof window === 'undefined') {
    return false;
  }
  return (
    localStorage.getItem(KOVAR_TERMS_STORAGE_KEY) === KOVAR_TERMS_VERSION
  );
}

export function acceptKovarTerms() {
  localStorage.setItem(KOVAR_TERMS_STORAGE_KEY, KOVAR_TERMS_VERSION);
}
