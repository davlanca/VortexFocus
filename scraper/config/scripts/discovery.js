(function() {
    const items = [];
    const seen = new Set();
    const loc = window.location;

    // Собираем абсолютно все ссылки на странице
    const allLinks = Array.from(document.querySelectorAll('a[href]'));

    allLinks.forEach(a => {
        try {
            const url = new URL(a.href);
            // Работаем только в рамках одного домена
            if (url.origin !== loc.origin) return;

            const path = url.pathname.toLowerCase();
            const parts = path.split('/').filter(Boolean);

            if (parts.length < 2) return;

            let type = '';
            let slug = '';

            // Оружие: /weapons/ak-47/ -> parts=['weapons', 'ak-47']
            if (parts[0] === 'weapons' && parts.length === 2) {
                type = 'weapon';
                slug = parts[1];
            }
            // Скины: /skins/ak-47-inheritance/ -> parts=['skins', 'ak-47-inheritance']
            else if (parts[0] === 'skins' && parts.length === 2) {
                type = 'skin';
                slug = parts[1];
            }

            // Исключаем системные страницы
            if (type && !['page', 'weapons', 'skins', 'compare'].includes(slug)) {
                if (!seen.has(path)) {
                    seen.add(path);
                    items.push({
                        slug: slug,
                        name: a.textContent.trim() || slug,
                        url: url.origin + path,
                        type: type
                    });
                }
            }
        } catch (e) {}
    });

    return {
        items: items,
        hasNextPage: false,
        nextPageUrl: null
    };
})();
