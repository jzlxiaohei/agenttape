import { useTranslation } from "react-i18next";
import { Languages } from "lucide-react";

// Pure view: toggles language via i18n. No business state.
export function LanguageToggle() {
  const { i18n, t } = useTranslation();
  const next = i18n.language.startsWith("zh") ? "en" : "zh";
  return (
    <button
      onClick={() => void i18n.changeLanguage(next)}
      className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
    >
      <Languages size={14} />
      {t("lang.toggle")}
    </button>
  );
}
