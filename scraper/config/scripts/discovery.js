(function() {
    const items = [];
    const seen = new Set();
    const loc = window.location;

    const targetPaths = [
        '/weapons/',
        '/skins/',
        '/gloves/',
        '/cases/',
        '/agents/',
        '/souvenir-packages/',
        '/collectible-pins/',
        '/patches/',
        '/sticker-capsules/',
        '/stickers/'
    ];

    let currentPath = loc.pathname.toLowerCase();
    if (!currentPath.endsWith('/')) {
        currentPath += '/';
    }

    const activeCategory = targetPaths
        .filter(p => currentPath.startsWith(p))
        .sort((a, b) => b.length - a.length)[0] || '/';

    /*
     * CSGO Database structure:
     *
     * /weapons/
     *     -> /weapons/talon-knife/
     *         -> /skins/talon-knife-blue-steel/
     *
     * /gloves/
     *     -> /gloves/sport-gloves/
     *         -> /skins/sport-gloves-...
     *
     * Therefore item pages are not always nested below the
     * category page URL.
     */
    function isAllowedPath(path) {
        // Never leave CSGO Database.
        if (!path || path === '/') {
            return false;
        }

        // Ignore pagination and utility pages.
        if (path.includes('/page/')) {
            return false;
        }

        if (path.includes('/compare/')) {
            return false;
        }

        if (path.includes('/search')) {
            return false;
        }

        // Do not return category root pages as items.
        if (targetPaths.includes(path)) {
            return false;
        }

        // Normal nested links.
        if (path.startsWith(activeCategory)) {
            return true;
        }

        /*
         * Weapon/glove category pages link to actual skin pages
         * under /skins/.
         */
        if (
            (activeCategory === '/weapons/' || activeCategory === '/gloves/') &&
            path.startsWith('/skins/')
        ) {
            return true;
        }

        /*
         * Other nested structures.
         */
        if (
            activeCategory === '/sticker-capsules/' &&
            path.startsWith('/stickers/')
        ) {
            return true;
        }

        if (
            activeCategory === '/collectible-pins/' &&
            path.startsWith('/collectible-pins/')
        ) {
            return true;
        }

        return false;
    }

    function getName(anchor) {
        const card =
            anchor.closest('.item-box') ||
            anchor.closest('.skin-box') ||
            anchor.closest('article') ||
            anchor.closest('li');

        let name =
            card?.querySelector('img[alt]')?.getAttribute('alt') ||
            card?.querySelector('.item-name')?.textContent ||
            card?.querySelector('h2')?.textContent ||
            card?.querySelector('h3')?.textContent ||
            anchor.textContent;

        if (!name) {
            return '';
        }

        name = name
            .replace(/\s+/g, ' ')
            .trim();

        /*
         * Remove common logo / decorative suffixes.
         */
        name = name
            .replace(/\s+logo$/i, '')
            .trim();

        return name;
    }

    document.querySelectorAll('a[href]').forEach(anchor => {
        try {
            const url = new URL(anchor.href, loc.origin);

            // Only same-origin links.
            if (url.origin !== loc.origin) {
                return;
            }

            let path = url.pathname.toLowerCase();

            if (!path.endsWith('/')) {
                path += '/';
            }

            if (!isAllowedPath(path)) {
                return;
            }

            /*
             * Only accept known CSGO Database content sections.
             */
            const type = path.split('/').filter(Boolean)[0];

            if (!targetPaths.some(p => p === `/${type}/`)) {
                return;
            }

            /*
             * Avoid generic navigation labels.
             */
            const name = getName(anchor);

            if (!name) {
                return;
            }

            if (
                /^(gloves|cases|stickers|patches|agents|pins|weapons|souvenir packages|souvenir packages)$/i.test(name)
            ) {
                return;
            }

            /*
             * Avoid accidentally treating marketplace links,
             * comparison links, etc. as database items.
             */
            if (
                url.hostname !== loc.hostname ||
                !url.pathname
            ) {
                return;
            }

            if (seen.has(url.href)) {
                return;
            }

            seen.add(url.href);

            items.push({
                slug: path.split('/').filter(Boolean).pop(),
                name: name,
                url: url.origin + path,
                type: type === 'skins' ? 'skin' : type
            });

        } catch (e) {
            // Ignore malformed links.
        }
    });

    return {
        items: items,
        hasNextPage: false,
        nextPageUrl: null
    };
})();
