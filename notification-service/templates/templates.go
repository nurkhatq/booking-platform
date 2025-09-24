package templates

import (
    "fmt"
)

var templates = map[string]map[string]string{
    "booking_confirmation": {
        "en": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Booking Confirmation</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #f9f9f9; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Booking Confirmed!</h1>
        </div>
        <div class="content">
            <p>Dear {{client_name}},</p>
            <p>Your appointment has been successfully confirmed. Here are your booking details:</p>
            
            <div class="details">
                <h3>Booking Details</h3>
                <p><strong>Business:</strong> {{business_name}}</p>
                <p><strong>Service:</strong> {{service_name}}</p>
                <p><strong>Master:</strong> {{master_name}}</p>
                <p><strong>Date:</strong> {{booking_date}}</p>
                <p><strong>Time:</strong> {{booking_time}}</p>
                <p><strong>Location:</strong> {{location_address}}</p>
                <p><strong>Confirmation Code:</strong> {{confirmation_code}}</p>
            </div>
            
            <p>Please arrive 10 minutes before your scheduled appointment time.</p>
            <p>If you need to cancel or reschedule, please contact us at least 2 hours in advance.</p>
        </div>
        <div class="footer">
            <p>Thank you for choosing {{business_name}}!</p>
        </div>
    </div>
</body>
</html>`,
        "ru": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Подтверждение бронирования</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #f9f9f9; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Бронирование подтверждено!</h1>
        </div>
        <div class="content">
            <p>Уважаемый(ая) {{client_name}},</p>
            <p>Ваша запись успешно подтверждена. Детали вашего бронирования:</p>
            
            <div class="details">
                <h3>Детали бронирования</h3>
                <p><strong>Бизнес:</strong> {{business_name}}</p>
                <p><strong>Услуга:</strong> {{service_name}}</p>
                <p><strong>Мастер:</strong> {{master_name}}</p>
                <p><strong>Дата:</strong> {{booking_date}}</p>
                <p><strong>Время:</strong> {{booking_time}}</p>
                <p><strong>Адрес:</strong> {{location_address}}</p>
                <p><strong>Код подтверждения:</strong> {{confirmation_code}}</p>
            </div>
            
            <p>Пожалуйста, приходите за 10 минут до назначенного времени.</p>
            <p>Для отмены или переноса записи свяжитесь с нами минимум за 2 часа.</p>
        </div>
        <div class="footer">
            <p>Спасибо за выбор {{business_name}}!</p>
        </div>
    </div>
</body>
</html>`,
        "kk": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Брондауды растау</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #f9f9f9; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Брондау расталды!</h1>
        </div>
        <div class="content">
            <p>Құрметті {{client_name}},</p>
            <p>Сіздің жазылуыңыз сәтті расталды. Брондау мәліметтері:</p>
            
            <div class="details">
                <h3>Брондау мәліметтері</h3>
                <p><strong>Бизнес:</strong> {{business_name}}</p>
                <p><strong>Қызмет:</strong> {{service_name}}</p>
                <p><strong>Шебер:</strong> {{master_name}}</p>
                <p><strong>Күні:</strong> {{booking_date}}</p>
                <p><strong>Уақыт:</strong> {{booking_time}}</p>
                <p><strong>Мекенжай:</strong> {{location_address}}</p>
                <p><strong>Растау коды:</strong> {{confirmation_code}}</p>
            </div>
            
            <p>Тағайындалған уақыттан 10 минут бұрын келіңіз.</p>
            <p>Жазылуды тоқтату немесе көшіру үшін кем дегенде 2 сағат бұрын хабарласыңыз.</p>
        </div>
        <div class="footer">
            <p>{{business_name}} таңдағаныңыз үшін рахмет!</p>
        </div>
    </div>
</body>
</html>`,
    },
    "booking_reminder": {
        "en": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Appointment Reminder</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #e3f2fd; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Appointment Reminder</h1>
        </div>
        <div class="content">
            <p>Dear {{client_name}},</p>
            <p>This is a friendly reminder about your upcoming appointment in {{hours_before}} hours.</p>
            
            <div class="details">
                <h3>Appointment Details</h3>
                <p><strong>Business:</strong> {{business_name}}</p>
                <p><strong>Service:</strong> {{service_name}}</p>
                <p><strong>Master:</strong> {{master_name}}</p>
                <p><strong>Date:</strong> {{booking_date}}</p>
                <p><strong>Time:</strong> {{booking_time}}</p>
                <p><strong>Location:</strong> {{location_address}}</p>
            </div>
            
            <p>Please arrive 10 minutes before your scheduled time.</p>
            <p>If you need to reschedule or cancel, please contact us as soon as possible.</p>
        </div>
        <div class="footer">
            <p>See you soon at {{business_name}}!</p>
        </div>
    </div>
</body>
</html>`,
        "ru": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Напоминание о приеме</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #e3f2fd; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Напоминание о приеме</h1>
        </div>
        <div class="content">
            <p>Уважаемый(ая) {{client_name}},</p>
            <p>Напоминаем о вашем предстоящем приеме через {{hours_before}} часов.</p>
            
            <div class="details">
                <h3>Детали приема</h3>
                <p><strong>Бизнес:</strong> {{business_name}}</p>
                <p><strong>Услуга:</strong> {{service_name}}</p>
                <p><strong>Мастер:</strong> {{master_name}}</p>
                <p><strong>Дата:</strong> {{booking_date}}</p>
                <p><strong>Время:</strong> {{booking_time}}</p>
                <p><strong>Адрес:</strong> {{location_address}}</p>
            </div>
            
            <p>Пожалуйста, приходите за 10 минут до назначенного времени.</p>
            <p>Если нужно перенести или отменить прием, свяжитесь с нами как можно скорее.</p>
        </div>
        <div class="footer">
            <p>До встречи в {{business_name}}!</p>
        </div>
    </div>
</body>
</html>`,
        "kk": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Прием туралы еске салу</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2196F3; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #e3f2fd; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Прием туралы еске салу</h1>
        </div>
        <div class="content">
            <p>Құрметті {{client_name}},</p>
            <p>{{hours_before}} сағаттан кейін приеміңіз туралы еске салу.</p>
            
            <div class="details">
                <h3>Прием мәліметтері</h3>
                <p><strong>Бизнес:</strong> {{business_name}}</p>
                <p><strong>Қызмет:</strong> {{service_name}}</p>
                <p><strong>Шебер:</strong> {{master_name}}</p>
                <p><strong>Күні:</strong> {{booking_date}}</p>
                <p><strong>Уақыт:</strong> {{booking_time}}</p>
                <p><strong>Мекенжай:</strong> {{location_address}}</p>
            </div>
            
            <p>Тағайындалған уақыттан 10 минут бұрын келіңіз.</p>
            <p>Приемді көшіру немесе тоқтату керек болса, мүмкіндігінше тез хабарласыңыз.</p>
        </div>
        <div class="footer">
            <p>{{business_name}}-те кездескенше!</p>
        </div>
    </div>
</body>
</html>`,
    },
    "booking_cancellation": {
        "en": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Booking Cancelled</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f44336; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .details { background: #ffebee; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Booking Cancelled</h1>
        </div>
        <div class="content">
            <p>Dear {{client_name}},</p>
            <p>We regret to inform you that your appointment has been cancelled.</p>
            
            <div class="details">
                <h3>Cancelled Appointment</h3>
                <p><strong>Business:</strong> {{business_name}}</p>
                <p><strong>Service:</strong> {{service_name}}</p>
                <p><strong>Date:</strong> {{booking_date}}</p>
                <p><strong>Time:</strong> {{booking_time}}</p>
                <p><strong>Reason:</strong> {{cancellation_reason}}</p>
            </div>
            
            <p>We apologize for any inconvenience this may cause.</p>
            <p>Please feel free to book a new appointment at your convenience.</p>
        </div>
        <div class="footer">
            <p>Thank you for your understanding - {{business_name}}</p>
        </div>
    </div>
</body>
</html>`,
        // Russian and Kazakh versions would follow the same pattern...
    },
}

func GetTemplate(templateName, language string) (string, bool) {
    if langTemplates, exists := templates[templateName]; exists {
        if template, exists := langTemplates[language]; exists {
            return template, true
        }
    }
    return "", false
}

func GetAvailableTemplates() []string {
    var templateNames []string
    for name := range templates {
        templateNames = append(templateNames, name)
    }
    return templateNames
}

func GetSupportedLanguages(templateName string) []string {
    if langTemplates, exists := templates[templateName]; exists {
        var languages []string
        for lang := range langTemplates {
            languages = append(languages, lang)
        }
        return languages
    }
    return []string{}
}