'use client';

import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useLanguage } from '@/lib/i18n';

export function LanguageToggle() {
  const { language, toggleLanguage, t } = useLanguage();
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant='secondary'
            size='sm'
            className='h-8 gap-1.5 px-2.5'
            onClick={toggleLanguage}
          />
        }
      >
        <Icons.language className='size-4' />
        <span className='text-xs font-semibold'>{language === 'en' ? 'EN' : 'SW'}</span>
        <span className='sr-only'>{t('toggleLanguage')}</span>
      </TooltipTrigger>
      <TooltipContent>{t('toggleLanguage')}</TooltipContent>
    </Tooltip>
  );
}
