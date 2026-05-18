from playwright.sync_api import sync_playwright


def wait_network(page):
    page.wait_for_load_state('networkidle')


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 390, "height": 844})

    page.goto('http://localhost:5173/?app=passenger')
    wait_network(page)
    page.locator('input[placeholder="手机号"]').fill('13800000001')
    page.locator('input[placeholder="验证码，演示固定 123456"]').fill('123456')
    page.get_by_role('button', name='登录').click()
    wait_network(page)
    page.get_by_role('button', name='呼叫快车').click()
    wait_network(page)
    passenger_text = page.text_content('body') or ''
    assert '我的订单' in passenger_text

    driver = browser.new_page(viewport={"width": 390, "height": 844})
    driver.goto('http://localhost:5173/?app=driver')
    wait_network(driver)
    driver_text = driver.text_content('body') or ''
    assert '司机端' in driver_text
    if driver.get_by_role('button', name='接单').count() > 0:
        driver.get_by_role('button', name='接单').first.click()
        wait_network(driver)

    admin = browser.new_page(viewport={"width": 1440, "height": 900})
    admin.goto('http://localhost:5173/?app=admin')
    wait_network(admin)
    admin_text = admin.text_content('body') or ''
    assert '管理后台' in admin_text
    assert '订单查询' in admin_text

    page.screenshot(path='/root/didi/docs/implementation/passenger-smoke.png', full_page=True)
    driver.screenshot(path='/root/didi/docs/implementation/driver-smoke.png', full_page=True)
    admin.screenshot(path='/root/didi/docs/implementation/admin-smoke.png', full_page=True)

    print('SMOKE_OK')
    browser.close()
